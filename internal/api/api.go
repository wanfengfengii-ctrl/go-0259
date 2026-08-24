// Package api exposes the JSON HTTP surface described by the interface section
// of the project document: stable error codes, deterministically sorted error
// details, audit queries, health checks, a controllable logical clock and
// device-script injection for deterministic testing.
package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"mycocycle-growbag-transfer-gate/internal/domain"
	"mycocycle-growbag-transfer-gate/internal/service"
)

// Server is the JSON HTTP API surface.
type Server struct {
	svc         *service.Service
	mux         *http.ServeMux
	handler     http.Handler
	logRequests bool
}

// Option customizes the API server.
type Option func(*Server)

// WithRequestLogging enables or disables per-request access logging.
func WithRequestLogging(on bool) Option {
	return func(s *Server) { s.logRequests = on }
}

// New constructs the API server and registers every documented route.
func New(svc *service.Service, opts ...Option) *Server {
	s := &Server{svc: svc, mux: http.NewServeMux(), logRequests: true}
	for _, o := range opts {
		o(s)
	}
	s.routes()
	s.handler = s.applyMiddleware(s.mux)
	return s
}

func (s *Server) routes() {
	s.mux.HandleFunc("POST /v1/tasks/lock", s.handleLock)
	s.mux.HandleFunc("POST /v1/tasks/{id}/sampling-confirmations", s.handleSamplingConfirmations)
	s.mux.HandleFunc("POST /v1/tasks/{id}/sample-seals", s.handleSampleSeals)
	s.mux.HandleFunc("POST /v1/tasks/{id}/occupancy/start", s.handleOccupancyStart)
	s.mux.HandleFunc("POST /v1/tasks/{id}/occupancy/move-rack", s.handleOccupancyMoveRack)
	s.mux.HandleFunc("POST /v1/tasks/{id}/observations", s.handleObservations)
	s.mux.HandleFunc("POST /v1/tasks/{id}/device-readings", s.handleDeviceReadings)
	s.mux.HandleFunc("POST /v1/tasks/{id}/contamination-rechecks", s.handleContaminationRechecks)
	s.mux.HandleFunc("POST /v1/tasks/{id}/reviews", s.handleReviews)
	s.mux.HandleFunc("POST /v1/tasks/{id}/finalize", s.handleFinalize)
	s.mux.HandleFunc("GET /v1/tasks", s.handleListTasks)
	s.mux.HandleFunc("GET /v1/tasks/{id}", s.handleGetTask)
	s.mux.HandleFunc("GET /v1/tasks/{id}/detail", s.handleGetTaskDetail)
	s.mux.HandleFunc("GET /v1/tasks/{id}/progress", s.handleGetTaskProgress)
	s.mux.HandleFunc("GET /v1/diagnostics", s.handleDiagnostics)
	s.mux.HandleFunc("GET /v1/catalog", s.handleCatalog)
	s.mux.HandleFunc("GET /v1/tasks/{id}/audit", s.handleGetAudit)
	s.mux.HandleFunc("GET /healthz", s.handleHealth)
	s.mux.HandleFunc("POST /v1/clock/advance", s.handleClockAdvance)
	s.mux.HandleFunc("POST /v1/devices/scripts", s.handleDeviceScript)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.handler.ServeHTTP(w, r)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if s.svc != nil {
		if err := s.svc.Healthy(); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "degraded", "error": err.Error()})
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError encodes a domain error with its stable code and sorted reasons.
func writeError(w http.ResponseWriter, err error) {
	var de *domain.Error
	if errors.As(err, &de) {
		writeJSON(w, httpStatus(de.Code), de)
		return
	}
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
}

func httpStatus(code domain.ErrorCode) int {
	switch code {
	case domain.CodeFinalStateRejected, domain.CodeGenerationConflict,
		domain.CodeIdempotencyConflict, domain.CodeResourceWindowConflict:
		return http.StatusConflict
	case domain.CodeDeviceFailure:
		return http.StatusBadGateway
	default:
		return http.StatusUnprocessableEntity
	}
}
