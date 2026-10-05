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

// handleSyncMasterCourses menangani POST /api/v1/master/courses/sync-jadwal
func (s *Server) handleSyncMasterCourses(w http.ResponseWriter, r *http.Request) {
	if s.masterController != nil {
		s.masterController.SyncCoursesFromJadwal(w, r)
		return
	}
	http.Error(w, "Master controller belum diinisialisasi", http.StatusInternalServerError)
}

// handleBulkCreateMasterCourses menangani POST /api/v1/master/courses/bulk
func (s *Server) handleBulkCreateMasterCourses(w http.ResponseWriter, r *http.Request) {
	if s.masterController != nil {
		s.masterController.BulkCreateCourses(w, r)
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

// handleGetMasterLecturers menangani GET /api/v1/master/lecturers
func (s *Server) handleGetMasterLecturers(w http.ResponseWriter, r *http.Request) {
	if s.masterController != nil {
		s.masterController.GetLecturers(w, r)
		return
	}
	http.Error(w, "Master controller belum diinisialisasi", http.StatusInternalServerError)
}

// handleCreateMasterLecturer menangani POST /api/v1/master/lecturers
func (s *Server) handleCreateMasterLecturer(w http.ResponseWriter, r *http.Request) {
	if s.masterController != nil {
		s.masterController.CreateLecturer(w, r)
		return
	}
	http.Error(w, "Master controller belum diinisialisasi", http.StatusInternalServerError)
}

// handlePatchMasterLecturer menangani PATCH /api/v1/master/lecturers/{id}
func (s *Server) handlePatchMasterLecturer(w http.ResponseWriter, r *http.Request) {
	if s.masterController != nil {
		s.masterController.PatchLecturer(w, r)
		return
	}
	http.Error(w, "Master controller belum diinisialisasi", http.StatusInternalServerError)
}

// handleSyncMasterLecturers menangani POST /api/v1/master/lecturers/sync-jadwal
func (s *Server) handleSyncMasterLecturers(w http.ResponseWriter, r *http.Request) {
	if s.masterController != nil {
		s.masterController.SyncLecturersFromJadwal(w, r)
		return
	}
	http.Error(w, "Master controller belum diinisialisasi", http.StatusInternalServerError)
}

// handleBulkCreateMasterLecturers menangani POST /api/v1/master/lecturers/bulk
func (s *Server) handleBulkCreateMasterLecturers(w http.ResponseWriter, r *http.Request) {
	if s.masterController != nil {
		s.masterController.BulkCreateLecturers(w, r)
		return
	}
	http.Error(w, "Master controller belum diinisialisasi", http.StatusInternalServerError)
}

// handleBulkCreateMasterRooms menangani POST /api/v1/master/rooms/bulk
func (s *Server) handleBulkCreateMasterRooms(w http.ResponseWriter, r *http.Request) {
	if s.masterController != nil {
		s.masterController.BulkCreateRooms(w, r)
		return
	}
	http.Error(w, "Master controller belum diinisialisasi", http.StatusInternalServerError)
}

// handleSyncMasterRooms menangani POST /api/v1/master/rooms/sync-jadwal
func (s *Server) handleSyncMasterRooms(w http.ResponseWriter, r *http.Request) {
	if s.masterController != nil {
		s.masterController.SyncRoomsFromJadwal(w, r)
		return
	}
	http.Error(w, "Master controller belum diinisialisasi", http.StatusInternalServerError)
}

// handleSyncAllMaster menangani POST /api/v1/master/sync-all
func (s *Server) handleSyncAllMaster(w http.ResponseWriter, r *http.Request) {
	if s.masterController != nil {
		s.masterController.SyncAllMasterFromJadwal(w, r)
		return
	}
	http.Error(w, "Master controller belum diinisialisasi", http.StatusInternalServerError)
}
