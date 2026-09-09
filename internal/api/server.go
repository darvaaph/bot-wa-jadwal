package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"bot-jadwal/internal/bot"
	"bot-jadwal/internal/schedule"
	"bot-jadwal/web"
)

// Server mengelola HTTP REST API untuk Web Admin Dashboard
type Server struct {
	httpServer   *http.Server
	botClient    *bot.BotClient
	classManager *schedule.ClassManager
}

// HealthResponse adalah payload untuk endpoint /api/health
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Uptime    string    `json:"uptime"`
}

// StatusResponse adalah payload telemetri untuk endpoint /api/status
type StatusResponse struct {
	Status        string    `json:"status"`
	Timestamp     time.Time `json:"timestamp"`
	BotConnection string    `json:"bot_connection"`
	TotalClasses  int       `json:"total_classes"`
	DefaultClass  string    `json:"default_class"`
	Classes       []string  `json:"classes"`
}

var startTime = time.Now()

// NewServer membuat instance baru HTTP API server dengan middleware CORS dan logging
func NewServer(addr string, botClient *bot.BotClient, classManager *schedule.ClassManager) *Server {
	mux := http.NewServeMux()

	s := &Server{
		botClient:    botClient,
		classManager: classManager,
	}

	// Registrasi Route API Scaffolding (Fase A)
	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("GET /api/status", s.handleStatus)

	// Fallback untuk route API yang belum diimplementasikan
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		s.writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "Endpoint belum tersedia (dijadwalkan pada Fase B)",
		})
	})

	// Menyajikan aset web statis (Dashboard Admin) dari web.Files embedded
	mux.Handle("/", http.FileServer(http.FS(web.Files)))

	handler := s.corsMiddleware(s.recoveryMiddleware(mux))

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return s
}

// handleHealth mengembalikan sinyal hidup (health check) server
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	resp := HealthResponse{
		Status:    "ok",
		Timestamp: time.Now(),
		Uptime:    time.Since(startTime).Round(time.Second).String(),
	}
	s.writeJSON(w, http.StatusOK, resp)
}

// handleStatus mengembalikan telemetri bot dan sistem kelas
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	botStatus := "uninitialized"
	if s.botClient != nil {
		botStatus = s.botClient.Status()
	}

	totalClasses := 0
	defaultClass := ""
	var classes []string
	if s.classManager != nil {
		classes = s.classManager.ListClasses()
		totalClasses = len(classes)
		defaultClass = s.classManager.GetDefaultClassID()
	}

	resp := StatusResponse{
		Status:        "ok",
		Timestamp:     time.Now(),
		BotConnection: botStatus,
		TotalClasses:  totalClasses,
		DefaultClass:  defaultClass,
		Classes:       classes,
	}
	s.writeJSON(w, http.StatusOK, resp)
}

// writeJSON adalah helper pengirim respon JSON seragam
func (s *Server) writeJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(data)
}

// corsMiddleware memungkinkan Web Dashboard (UI/UX) diakses lintas port saat masa pengembangan
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// recoveryMiddleware menangani panic HTTP agar server web tidak crash
func (s *Server) recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				fmt.Printf("⚠️ [HTTP Panic] %v\n", rec)
				s.writeJSON(w, http.StatusInternalServerError, map[string]string{
					"error": "Terjadi kesalahan internal server",
				})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// Start menjalankan HTTP Server di background goroutine
func (s *Server) Start() error {
	fmt.Printf("🌐 [Web API] Server REST API aktif di http://localhost%s\n", s.httpServer.Addr)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("⚠️ [Web API] Server berhenti dengan pesan: %v\n", err)
		}
	}()
	return nil
}

// Shutdown mematikan HTTP server secara anggun (graceful shutdown)
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}
