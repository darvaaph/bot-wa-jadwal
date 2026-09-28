package api

import (
	"fmt"
	"net/http"
)

// handleNotImplemented mengembalikan respon HTTP 501 terstruktur untuk fitur yang dijadwalkan pada v1.1+
func (s *Server) handleNotImplemented(featureName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		msg := fmt.Sprintf("Tahap v1.1: %s", featureName)
		s.writeV1Error(w, http.StatusNotImplemented, CodeNotImplemented, msg)
	}
}

// Deferred handlers untuk rute-rute tahap v1.1+ sesuai docs/api/API_V1.md §7
func (s *Server) handleGetRoomCandidates(w http.ResponseWriter, r *http.Request) {
	s.handleNotImplemented("Kandidat Ruangan Kosong")(w, r)
}

func (s *Server) handleCreateRoomConfirmation(w http.ResponseWriter, r *http.Request) {
	s.handleNotImplemented("Konfirmasi Ruangan TU")(w, r)
}

func (s *Server) handleGetNotifications(w http.ResponseWriter, r *http.Request) {
	s.handleNotImplemented("Daftar Antrean Notifikasi")(w, r)
}

func (s *Server) handleRetryNotification(w http.ResponseWriter, r *http.Request) {
	s.handleNotImplemented("Coba Ulang Pengiriman Notifikasi")(w, r)
}

func (s *Server) handleGetAuditLogs(w http.ResponseWriter, r *http.Request) {
	s.handleNotImplemented("Log Audit Sistem")(w, r)
}

func (s *Server) handleCreateBackup(w http.ResponseWriter, r *http.Request) {
	s.handleNotImplemented("Pencadangan Database")(w, r)
}

func (s *Server) handleRestoreBackup(w http.ResponseWriter, r *http.Request) {
	s.handleNotImplemented("Pemulihan Cadangan Database")(w, r)
}

func (s *Server) handleGetAdminStatus(w http.ResponseWriter, r *http.Request) {
	s.handleNotImplemented("Status Telemetri Admin")(w, r)
}

func (s *Server) handleAdminSuspendUser(w http.ResponseWriter, r *http.Request) {
	s.handleNotImplemented("Penangguhan Akun Pengguna")(w, r)
}

func (s *Server) handleAdminRecoverUser(w http.ResponseWriter, r *http.Request) {
	s.handleNotImplemented("Pemulihan Akun Pengguna")(w, r)
}

func (s *Server) handleSemesterImportValidate(w http.ResponseWriter, r *http.Request) {
	s.handleNotImplemented("Validasi Impor Kurikulum Batch")(w, r)
}

func (s *Server) handleSemesterImportApply(w http.ResponseWriter, r *http.Request) {
	s.handleNotImplemented("Penerapan Impor Kurikulum Batch")(w, r)
}
