package api

import (
	"bot-jadwal/internal/api/middleware"
	"bot-jadwal/internal/ratelimit"
)

// SetRateLimiter memasang limiter terpusat pada server.
func (s *Server) SetRateLimiter(limiter *ratelimit.Service) {
	s.limiter = limiter
	if s.rlManager != nil {
		s.rlManager.SetLimiter(limiter)
	}
	if s.portalService != nil && limiter != nil {
		s.portalService.SetRateLimiter(middleware.PortalLimiterAdapter{Svc: limiter})
	}
}

// buildLimiter membangun limiter terpusat di atas database v1.
func (s *Server) buildLimiter() {
	if s.v1DB == nil {
		return
	}
	if limiter, err := ratelimit.NewService(s.v1DB, s.authHashKey, nil); err == nil {
		s.SetRateLimiter(limiter)
	}
}
