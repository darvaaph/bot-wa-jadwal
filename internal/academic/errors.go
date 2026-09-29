package academic

import "errors"

// Standar domain errors untuk paket academic.
var (
	ErrNotFound     = errors.New("data tidak ditemukan")
	ErrInvalidInput = errors.New("input tidak valid")
	ErrConflict     = errors.New("data bertabrakan atau kode sudah digunakan")
	ErrInvalidState = errors.New("status tidak memungkinkan operasi ini")
	ErrForbidden    = errors.New("tindakan tidak tersedia pada cakupan aktif")
	ErrVersion      = errors.New("versi data sudah berubah, muat ulang sebelum menyimpan")
)
