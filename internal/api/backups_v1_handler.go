package api

import (
	"net/http"

	v1 "bot-jadwal/internal/api/v1"
)

// Alias types untuk backward-compatibility
type BackupRequest = v1.BackupRequest
type BackupResponseItem = v1.BackupResponseItem
type RestoreRequest = v1.RestoreRequest

// handleCreateBackup menangani POST /api/v1/backups
func (s *Server) handleCreateBackup(w http.ResponseWriter, r *http.Request) {
	if s.adminController != nil {
		s.adminController.CreateBackup(w, r)
		return
	}
	http.Error(w, "Admin controller belum diinisialisasi", http.StatusInternalServerError)
}

// handleRestoreBackup menangani POST /api/v1/restores
func (s *Server) handleRestoreBackup(w http.ResponseWriter, r *http.Request) {
	if s.adminController != nil {
		s.adminController.RestoreBackup(w, r)
		return
	}
	http.Error(w, "Admin controller belum diinisialisasi", http.StatusInternalServerError)
}
