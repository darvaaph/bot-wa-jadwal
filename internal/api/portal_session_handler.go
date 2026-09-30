package api

import (
	"net/http"

	v1 "bot-jadwal/internal/api/v1"
)

type createPortalSessionRequest = v1.CreatePortalSessionRequest
type rotatePortalCodeRequest = v1.RotatePortalCodeRequest

func (s *Server) handleCreatePortalSession(w http.ResponseWriter, r *http.Request) {
	if s.portalController != nil {
		s.portalController.CreateSession(w, r)
		return
	}
	v1.NewPortalController(s.v1DB, s.portalService, s.rlManager, s.secManager).CreateSession(w, r)
}

func (s *Server) handleRotatePortalCode(w http.ResponseWriter, r *http.Request) {
	if s.portalController != nil {
		s.portalController.RotateCode(w, r)
		return
	}
	v1.NewPortalController(s.v1DB, s.portalService, s.rlManager, s.secManager).RotateCode(w, r)
}

func (s *Server) handleGetClassSettings(w http.ResponseWriter, r *http.Request) {
	if s.portalController != nil {
		s.portalController.GetClassSettings(w, r)
		return
	}
	v1.NewPortalController(s.v1DB, s.portalService, s.rlManager, s.secManager).GetClassSettings(w, r)
}

func (s *Server) handleSetPortalMode(w http.ResponseWriter, r *http.Request) {
	if s.portalController != nil {
		s.portalController.SetPortalMode(w, r)
		return
	}
	v1.NewPortalController(s.v1DB, s.portalService, s.rlManager, s.secManager).SetPortalMode(w, r)
}
