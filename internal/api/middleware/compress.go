package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"sync"
)

var gzipWriterPool = sync.Pool{
	New: func() any {
		w, _ := gzip.NewWriterLevel(io.Discard, gzip.DefaultCompression)
		return w
	},
}

type gzipResponseWriter struct {
	http.ResponseWriter
	writer       *gzip.Writer
	wroteHeader  bool
	isCompressed bool
}

func (w *gzipResponseWriter) WriteHeader(statusCode int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true

	// Jangan kompres jika status no-content atau not-modified
	if statusCode == http.StatusNoContent || statusCode == http.StatusNotModified {
		w.isCompressed = false
		w.ResponseWriter.WriteHeader(statusCode)
		return
	}

	contentType := w.Header().Get("Content-Type")
	if isCompressibleContentType(contentType) {
		w.isCompressed = true
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Add("Vary", "Accept-Encoding")
		w.Header().Del("Content-Length") // Content-Length berubah setelah kompresi
	} else {
		w.isCompressed = false
	}

	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		if w.Header().Get("Content-Type") == "" {
			w.Header().Set("Content-Type", http.DetectContentType(b))
		}
		w.WriteHeader(http.StatusOK)
	}

	if w.isCompressed && w.writer != nil {
		return w.writer.Write(b)
	}
	return w.ResponseWriter.Write(b)
}

func (w *gzipResponseWriter) Flush() {
	if w.isCompressed && w.writer != nil {
		_ = w.writer.Flush()
	}
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// Gzip mengompresi payload respons jika klien mendukung header Accept-Encoding: gzip.
func Gzip(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") ||
			r.Header.Get("Upgrade") != "" ||
			r.Method == http.MethodHead {
			next.ServeHTTP(w, r)
			return
		}

		gz := gzipWriterPool.Get().(*gzip.Writer)
		defer gzipWriterPool.Put(gz)
		gz.Reset(w)

		gzw := &gzipResponseWriter{
			ResponseWriter: w,
			writer:         gz,
		}

		defer func() {
			if gzw.isCompressed {
				_ = gz.Close()
			}
		}()

		next.ServeHTTP(gzw, r)
	})
}

func isCompressibleContentType(ct string) bool {
	ct = strings.ToLower(ct)
	return strings.HasPrefix(ct, "text/") ||
		strings.Contains(ct, "application/javascript") ||
		strings.Contains(ct, "application/json") ||
		strings.Contains(ct, "application/manifest+json") ||
		strings.Contains(ct, "image/svg+xml") ||
		strings.Contains(ct, "application/xml")
}
