package api

import (
	"net/http"

	"bot-jadwal/internal/api/common"
)

// Standar kode error API v1 sesuai docs/api/API_V1.md §0
const (
	CodeUnauthenticated = common.CodeUnauthenticated // 401
	CodeForbidden       = common.CodeForbidden       // 403
	CodeNotFound        = common.CodeNotFound        // 404
	CodeValidation      = common.CodeValidation      // 422
	CodeVersionConflict = common.CodeVersionConflict // 409
	CodeGoneArchived    = common.CodeGoneArchived    // 410
	CodeNotImplemented  = common.CodeNotImplemented  // 501
	CodeTooManyRequests = common.CodeTooManyRequests // 429
	CodeServiceDown     = common.CodeServiceDown     // 503
)

// V1Response adalah envelope standar untuk seluruh respons sukses API v1
type V1Response = common.V1Response

// V1ErrorResponse adalah envelope standar untuk seluruh respons error API v1
type V1ErrorResponse = common.V1ErrorResponse

// V1ErrorDetail memuat kode error terstruktur dan pesan ramah pengguna
type V1ErrorDetail = common.V1ErrorDetail

// writeV1Success mengirimkan respons sukses dengan status 200/201 dan envelope {status: "success", data: ...}
func (s *Server) writeV1Success(w http.ResponseWriter, statusCode int, data any, meta ...any) {
	common.WriteV1Success(w, statusCode, data, meta...)
}

// writeV1Error mengirimkan respons error terstandarisasi sesuai spesifikasi API v1
func (s *Server) writeV1Error(w http.ResponseWriter, httpStatus int, code string, message string, details ...any) {
	common.WriteV1Error(w, httpStatus, code, message, details...)
}
