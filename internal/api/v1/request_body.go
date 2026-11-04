package v1

import (
	"errors"
	"io"
	"net/http"

	"bot-jadwal/internal/api/common"
)

const maxBufferedJSONBytes = 10 * 1024 * 1024

func readBufferedJSON(w http.ResponseWriter, r *http.Request) ([]byte, bool) {
	body := http.MaxBytesReader(w, r.Body, maxBufferedJSONBytes)
	defer body.Close()
	data, err := io.ReadAll(body)
	if err == nil {
		return data, true
	}
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		common.WriteV1Error(w, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "Payload melebihi batas 10 MiB")
	} else {
		common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, "Gagal membaca payload request")
	}
	return nil, false
}
