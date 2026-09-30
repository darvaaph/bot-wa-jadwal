package api

import (
	"net/http"
)

// handleGetMasterRooms menangani GET /api/v1/master/rooms
func (s *Server) handleGetMasterRooms(w http.ResponseWriter, r *http.Request) {
	if s.masterController != nil {
		s.masterController.GetRooms(w, r)
		return
	}
	http.Error(w, "Master controller belum diinisialisasi", http.StatusInternalServerError)
}

// handleCreateMasterRoom menangani POST /api/v1/master/rooms
func (s *Server) handleCreateMasterRoom(w http.ResponseWriter, r *http.Request) {
	if s.masterController != nil {
		s.masterController.CreateRoom(w, r)
		return
	}
	http.Error(w, "Master controller belum diinisialisasi", http.StatusInternalServerError)
}

// handlePatchMasterRoom menangani PATCH /api/v1/master/rooms/{id}
func (s *Server) handlePatchMasterRoom(w http.ResponseWriter, r *http.Request) {
	if s.masterController != nil {
		s.masterController.PatchRoom(w, r)
		return
	}
	http.Error(w, "Master controller belum diinisialisasi", http.StatusInternalServerError)
}

// handleGetMasterCourses menangani GET /api/v1/master/courses
func (s *Server) handleGetMasterCourses(w http.ResponseWriter, r *http.Request) {
	if s.masterController != nil {
		s.masterController.GetCourses(w, r)
		return
	}
	http.Error(w, "Master controller belum diinisialisasi", http.StatusInternalServerError)
}

// handleCreateMasterCourse menangani POST /api/v1/master/courses
func (s *Server) handleCreateMasterCourse(w http.ResponseWriter, r *http.Request) {
	if s.masterController != nil {
		s.masterController.CreateCourse(w, r)
		return
	}
	http.Error(w, "Master controller belum diinisialisasi", http.StatusInternalServerError)
}

// handlePatchMasterCourse menangani PATCH /api/v1/master/courses/{id}
func (s *Server) handlePatchMasterCourse(w http.ResponseWriter, r *http.Request) {
	if s.masterController != nil {
		s.masterController.PatchCourse(w, r)
		return
	}
	http.Error(w, "Master controller belum diinisialisasi", http.StatusInternalServerError)
}
