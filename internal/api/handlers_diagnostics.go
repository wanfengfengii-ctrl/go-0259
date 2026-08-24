package api

import (
	"net/http"

	"mycocycle-growbag-transfer-gate/internal/domain"
)

// handleGetTaskDetail returns the full task read model including leases, cells,
// readings, evidence, reviews and any terminal decision.
func (s *Server) handleGetTaskDetail(w http.ResponseWriter, r *http.Request) {
	id := domain.TaskID(r.PathValue("id"))
	res, err := s.svc.GetTaskDetail(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// handleDiagnostics returns the operational report (recovery counts, per-state
// task counts, task listing and pending device retries).
func (s *Server) handleDiagnostics(w http.ResponseWriter, r *http.Request) {
	report, err := s.svc.Report()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, report)
}

// handleGetTaskProgress returns the task progress view.
func (s *Server) handleGetTaskProgress(w http.ResponseWriter, r *http.Request) {
	id := domain.TaskID(r.PathValue("id"))
	res, err := s.svc.Progress(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// handleListTasks returns the compact task listing, optionally filtered by
// bag batch.
func (s *Server) handleListTasks(w http.ResponseWriter, r *http.Request) {
	batch := r.URL.Query().Get("batch")
	var (
		res any
		err error
	)
	if batch != "" {
		res, err = s.svc.TasksByBatch(batch)
	} else {
		res, err = s.svc.ListTaskSummaries()
	}
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// handleCatalog returns the reference catalog used for lock validation.
func (s *Server) handleCatalog(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.svc.CatalogSummary())
}
