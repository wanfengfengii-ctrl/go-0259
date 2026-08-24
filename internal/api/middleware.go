package api

import (
	"log"
	"net/http"
	"runtime/debug"
	"time"
)

// statusRecorder captures the response status code for request logging.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// withLogging wraps a handler with structured request logging.
func (s *Server) withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		if s.logRequests {
			log.Printf("http %s %s -> %d (%s)", r.Method, r.URL.Path, rec.status, time.Since(start))
		}
	})
}

// recoverMiddleware converts a panic into a stable 500 response instead of
// terminating the process.
func recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("panic serving %s %s: %v\n%s", r.Method, r.URL.Path, rec, debug.Stack())
				writeJSON(w, http.StatusInternalServerError, map[string]string{
					"error": "internal server error",
				})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// applyMiddleware wraps the root mux with the logging and panic-recovery
// middleware, returning the outermost handler.
func (s *Server) applyMiddleware(h http.Handler) http.Handler {
	return s.withLogging(recoverMiddleware(h))
}
