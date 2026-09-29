package api

import (
	"net/http"

	"bot-jadwal/internal/api/legacy"
)

// ClassesResponse adalah format balasan untuk endpoint GET /api/classes
type ClassesResponse = legacy.ClassesResponse

// ClassesDataResponse adalah payload data kelas yang terdaftar
type ClassesDataResponse = legacy.ClassesDataResponse

// ScheduleItemResponse merepresentasikan entri jadwal perkuliahan individual untuk respons API
type ScheduleItemResponse = legacy.ScheduleItemResponse

// ScheduleResponse adalah format balasan untuk endpoint GET /api/schedule
type ScheduleResponse = legacy.ScheduleResponse

// handleClasses menyajikan daftar seluruh kode kelas kanonikal beserta kelas default (Legacy Shim)
func (s *Server) handleClasses(w http.ResponseWriter, r *http.Request) {
	if s.legacyHandler != nil {
		s.legacyHandler.HandleClasses(w, r)
		return
	}
	legacy.NewHandler(s.classManager, s.taskManager, s.v1DB).HandleClasses(w, r)
}

// handleSchedule menyajikan jadwal perkuliahan berdasarkan kelas dan filter hari (Legacy Shim)
func (s *Server) handleSchedule(w http.ResponseWriter, r *http.Request) {
	if s.legacyHandler != nil {
		s.legacyHandler.HandleSchedule(w, r)
		return
	}
	legacy.NewHandler(s.classManager, s.taskManager, s.v1DB).HandleSchedule(w, r)
}
