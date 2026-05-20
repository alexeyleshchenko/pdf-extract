package httpserver

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

func isHealthPath(path string) bool {
	return path == "/health" || path == "/v1/health"
}

// AccessLog logs one line per HTTP request (except health checks at Info level).
func AccessLog(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			elapsed := time.Since(start)

			if log == nil {
				return
			}
			if isHealthPath(r.URL.Path) {
				log.Debug("http_request",
					"method", r.Method,
					"path", r.URL.Path,
					"status", ww.Status(),
					"bytes", ww.BytesWritten(),
					"duration_ms", elapsed.Milliseconds(),
					"request_id", middleware.GetReqID(r.Context()),
					"remote_addr", r.RemoteAddr,
				)
				return
			}
			log.Info("http_request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"bytes", ww.BytesWritten(),
				"duration_ms", elapsed.Milliseconds(),
				"request_id", middleware.GetReqID(r.Context()),
				"remote_addr", r.RemoteAddr,
			)
		})
	}
}
