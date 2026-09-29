package api

import (
	"encoding/json"
	"net/http"
)

// Standar kode error API v1 sesuai docs/api/API_V1.md §0
const (
	CodeUnauthenticated = "UNAUTHENTICATED"     // 401
	CodeForbidden       = "FORBIDDEN"           // 403
	CodeNotFound        = "NOT_FOUND"           // 404
	CodeValidation      = "VALIDATION"          // 422
	CodeVersionConflict = "VERSION_CONFLICT"    // 409
	CodeGoneArchived    = "GONE_ARCHIVED"       // 410
	CodeNotImplemented  = "NOT_IMPLEMENTED"     // 501
	CodeTooManyRequests = "TOO_MANY_REQUESTS"   // 429
	CodeServiceDown     = "SERVICE_UNAVAILABLE" // 503
)

// V1Response adalah envelope standar untuk seluruh respons sukses API v1
type V1Response struct {
	Status string `json:"status"`
	Data   any    `json:"data"`
	Meta   any    `json:"meta,omitempty"`
}

// V1ErrorDetail adalah envelope standar untuk seluruh respons error API v1
type V1ErrorResponse struct {
	Status string        `json:"status"`
	Error  V1ErrorDetail `json:"error"`
}

// V1ErrorDetail memuat kode error terstruktur dan pesan ramah pengguna
type V1ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// writeV1Success mengirimkan respons sukses dengan status 200/201 dan envelope {status: "success", data: ...}
func (s *Server) writeV1Success(w http.ResponseWriter, statusCode int, data any, meta ...any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)

	resp := V1Response{
		Status: "success",
		Data:   data,
	}
	if len(meta) > 0 && meta[0] != nil {
		resp.Meta = meta[0]
	}

	_ = json.NewEncoder(w).Encode(resp)
}

// writeV1Error mengirimkan respons error terstandarisasi sesuai spesifikasi API v1
func (s *Server) writeV1Error(w http.ResponseWriter, httpStatus int, code string, message string, details ...any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(httpStatus)

	resp := V1ErrorResponse{
		Status: "error",
		Error: V1ErrorDetail{
			Code:    code,
			Message: message,
		},
	}
	if len(details) > 0 && details[0] != nil {
		resp.Error.Details = details[0]
	}

	_ = json.NewEncoder(w).Encode(resp)
}
