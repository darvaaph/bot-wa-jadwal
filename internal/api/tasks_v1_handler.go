package api

import (
	"net/http"

	v1 "bot-jadwal/internal/api/v1"
)

// Alias types untuk backwards compatibility
type CreateTaskV1Request = v1.CreateTaskV1Request
type PatchTaskRequest = v1.PatchTaskRequest
type ReviewTaskRequest = v1.ReviewTaskRequest
type TaskStateRequest = v1.TaskStateRequest

func (s *Server) handleGetV1Tasks(w http.ResponseWriter, r *http.Request) {
	if s.taskController != nil {
		s.taskController.GetTasks(w, r)
		return
	}
	v1.NewTaskController(s.v1DB).GetTasks(w, r)
}

func (s *Server) handleCreateV1Task(w http.ResponseWriter, r *http.Request) {
	if s.taskController != nil {
		s.taskController.CreateTask(w, r)
		return
	}
	v1.NewTaskController(s.v1DB).CreateTask(w, r)
}

func (s *Server) handleGetV1TaskDetail(w http.ResponseWriter, r *http.Request) {
	if s.taskController != nil {
		s.taskController.GetTaskDetail(w, r)
		return
	}
	v1.NewTaskController(s.v1DB).GetTaskDetail(w, r)
}

func (s *Server) handlePatchV1Task(w http.ResponseWriter, r *http.Request) {
	if s.taskController != nil {
		s.taskController.PatchTask(w, r)
		return
	}
	v1.NewTaskController(s.v1DB).PatchTask(w, r)
}

func (s *Server) handleReviewV1Task(w http.ResponseWriter, r *http.Request) {
	if s.taskController != nil {
		s.taskController.ReviewTask(w, r)
		return
	}
	v1.NewTaskController(s.v1DB).ReviewTask(w, r)
}

func (s *Server) handleCompleteV1Task(w http.ResponseWriter, r *http.Request) {
	if s.taskController != nil {
		s.taskController.CompleteTask(w, r)
		return
	}
	v1.NewTaskController(s.v1DB).CompleteTask(w, r)
}

func (s *Server) handleArchiveV1Task(w http.ResponseWriter, r *http.Request) {
	if s.taskController != nil {
		s.taskController.ArchiveTask(w, r)
		return
	}
	v1.NewTaskController(s.v1DB).ArchiveTask(w, r)
}

func (s *Server) handleRestoreV1Task(w http.ResponseWriter, r *http.Request) {
	if s.taskController != nil {
		s.taskController.RestoreTask(w, r)
		return
	}
	v1.NewTaskController(s.v1DB).RestoreTask(w, r)
}
