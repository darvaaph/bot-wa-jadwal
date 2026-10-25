package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGzipMiddleware_CompressedWhenSupported(t *testing.T) {
	samplePayload := strings.Repeat("body { color: #3965FB; margin: 0; padding: 0; }\n", 50)

	handler := Gzip(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		_, _ = w.Write([]byte(samplePayload))
	}))

	req := httptest.NewRequest("GET", "/css/style.css", nil)
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("Content-Encoding harus 'gzip', didapat: %q", rec.Header().Get("Content-Encoding"))
	}

	gzReader, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatalf("Gagal membaca respons terkompresi gzip: %v", err)
	}
	defer gzReader.Close()

	decompressed, err := io.ReadAll(gzReader)
	if err != nil {
		t.Fatalf("Gagal mendekompresi payload: %v", err)
	}

	if string(decompressed) != samplePayload {
		t.Fatalf("Payload setelah dekompresi tidak sesuai")
	}

	if rec.Body.Len() >= len(samplePayload) {
		t.Errorf("Payload terkompresi (%d) harus lebih kecil dari asli (%d)", rec.Body.Len(), len(samplePayload))
	}
}

func TestGzipMiddleware_IgnoredWhenNoHeader(t *testing.T) {
	samplePayload := "Halo dunia tanpa kompresi"

	handler := Gzip(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(samplePayload))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Encoding") != "" {
		t.Errorf("Content-Encoding harus kosong jika klien tidak meminta gzip")
	}
	if rec.Body.String() != samplePayload {
		t.Errorf("Isi body salah: %s", rec.Body.String())
	}
}

func TestGzipMiddleware_NonCompressibleTypeNotCompressed(t *testing.T) {
	binaryPayload := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A} // PNG magic bytes

	handler := Gzip(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(binaryPayload)
	}))

	req := httptest.NewRequest("GET", "/icon.png", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Encoding") != "" {
		t.Errorf("image/png tidak boleh dikompresi gzip")
	}
	if !bytes.Equal(rec.Body.Bytes(), binaryPayload) {
		t.Errorf("Binary payload tidak boleh berubah")
	}
}
