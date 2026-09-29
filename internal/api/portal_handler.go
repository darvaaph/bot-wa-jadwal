package api

import (
	"net/http"

	v1 "bot-jadwal/internal/api/v1"
)

func (s *Server) handlePortalSummary(w http.ResponseWriter, r *http.Request) {
	if s.portalController != nil {
		s.portalController.Summary(w, r)
		return
	}
	v1.NewPortalController(s.v1DB, s.portalService, s.rlManager, s.secManager).Summary(w, r)
}

func (s *Server) handlePortalSchedule(w http.ResponseWriter, r *http.Request) {
	if s.portalController != nil {
		s.portalController.Schedule(w, r)
		return
	}
	v1.NewPortalController(s.v1DB, s.portalService, s.rlManager, s.secManager).Schedule(w, r)
}

func (s *Server) handlePortalTasks(w http.ResponseWriter, r *http.Request) {
	if s.portalController != nil {
		s.portalController.Tasks(w, r)
		return
	}
	v1.NewPortalController(s.v1DB, s.portalService, s.rlManager, s.secManager).Tasks(w, r)
}

func (s *Server) handlePortalTaskDetail(w http.ResponseWriter, r *http.Request) {
	if s.portalController != nil {
		s.portalController.TaskDetail(w, r)
		return
	}
	v1.NewPortalController(s.v1DB, s.portalService, s.rlManager, s.secManager).TaskDetail(w, r)
}

func (s *Server) handlePortalChanges(w http.ResponseWriter, r *http.Request) {
	if s.portalController != nil {
		s.portalController.Changes(w, r)
		return
	}
	v1.NewPortalController(s.v1DB, s.portalService, s.rlManager, s.secManager).Changes(w, r)
}

func (s *Server) handlePortalMaterials(w http.ResponseWriter, r *http.Request) {
	if s.portalController != nil {
		s.portalController.Materials(w, r)
		return
	}
	v1.NewPortalController(s.v1DB, s.portalService, s.rlManager, s.secManager).Materials(w, r)
}

func extractPortalToken(r *http.Request) string {
	return v1.ExtractPortalToken(r)
}

func (s *Server) getOfferingLecturers(offeringID int64) []string {
	return v1.GetOfferingLecturers(s.v1DB, offeringID)
}
