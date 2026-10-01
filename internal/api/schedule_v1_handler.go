package api

import (
	"net/http"

	v1 "bot-jadwal/internal/api/v1"
)

// Alias types untuk backwards compatibility
type CreatePatternRequest = v1.CreatePatternRequest
type CreateTeachingEventRequest = v1.CreateTeachingEventRequest
type PublishEventRequest = v1.PublishEventRequest
type RevokeEventRequest = v1.RevokeEventRequest
type PatchPatternRequest = v1.PatchPatternRequest
type ParticipationRequest = v1.ParticipationRequest

func (s *Server) handleGetV1Patterns(w http.ResponseWriter, r *http.Request) {
	if s.scheduleController != nil {
		s.scheduleController.GetPatterns(w, r)
		return
	}
	v1.NewScheduleController(s.v1DB).GetPatterns(w, r)
}

func (s *Server) handleCreateV1Pattern(w http.ResponseWriter, r *http.Request) {
	if s.scheduleController != nil {
		s.scheduleController.CreatePattern(w, r)
		return
	}
	v1.NewScheduleController(s.v1DB).CreatePattern(w, r)
}

func (s *Server) handlePatchV1Pattern(w http.ResponseWriter, r *http.Request) {
	if s.scheduleController != nil {
		s.scheduleController.PatchPattern(w, r)
		return
	}
	v1.NewScheduleController(s.v1DB).PatchPattern(w, r)
}

func (s *Server) handleCreateV1TeachingEvent(w http.ResponseWriter, r *http.Request) {
	if s.scheduleController != nil {
		s.scheduleController.CreateTeachingEvent(w, r)
		return
	}
	v1.NewScheduleController(s.v1DB).CreateTeachingEvent(w, r)
}

func (s *Server) handleGetV1TeachingEvents(w http.ResponseWriter, r *http.Request) {
	if s.scheduleController != nil {
		s.scheduleController.GetTeachingEvents(w, r)
		return
	}
	v1.NewScheduleController(s.v1DB).GetTeachingEvents(w, r)
}

func (s *Server) handlePreviewV1TeachingEvent(w http.ResponseWriter, r *http.Request) {
	if s.scheduleController != nil {
		s.scheduleController.PreviewTeachingEvent(w, r)
		return
	}
	v1.NewScheduleController(s.v1DB).PreviewTeachingEvent(w, r)
}

func (s *Server) handlePublishV1TeachingEvent(w http.ResponseWriter, r *http.Request) {
	if s.scheduleController != nil {
		s.scheduleController.PublishTeachingEvent(w, r)
		return
	}
	v1.NewScheduleController(s.v1DB).PublishTeachingEvent(w, r)
}

func (s *Server) handleRevokeV1TeachingEvent(w http.ResponseWriter, r *http.Request) {
	if s.scheduleController != nil {
		s.scheduleController.RevokeTeachingEvent(w, r)
		return
	}
	v1.NewScheduleController(s.v1DB).RevokeTeachingEvent(w, r)
}

func (s *Server) handleParticipationV1TeachingEvent(w http.ResponseWriter, r *http.Request) {
	if s.scheduleController != nil {
		s.scheduleController.ParticipationTeachingEvent(w, r)
		return
	}
	v1.NewScheduleController(s.v1DB).ParticipationTeachingEvent(w, r)
}

func (s *Server) handleDeleteV1Pattern(w http.ResponseWriter, r *http.Request) {
	if s.scheduleController != nil {
		s.scheduleController.DeletePattern(w, r)
		return
	}
	v1.NewScheduleController(s.v1DB).DeletePattern(w, r)
}

func (s *Server) handleDeleteV1TeachingEvent(w http.ResponseWriter, r *http.Request) {
	if s.scheduleController != nil {
		s.scheduleController.DeleteTeachingEvent(w, r)
		return
	}
	v1.NewScheduleController(s.v1DB).DeleteTeachingEvent(w, r)
}
