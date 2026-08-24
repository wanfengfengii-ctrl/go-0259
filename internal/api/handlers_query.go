package api

import (
	"net/http"

	"mycocycle-growbag-transfer-gate/internal/domain"
)

// handleGetTask returns the task read model.
func (s *Server) handleGetTask(w http.ResponseWriter, r *http.Request) {
	id := domain.TaskID(r.PathValue("id"))
	res, err := s.svc.GetTask(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// handleGetAudit returns the task audit trail, optionally filtered by action.
func (s *Server) handleGetAudit(w http.ResponseWriter, r *http.Request) {
	id := domain.TaskID(r.PathValue("id"))
	action := r.URL.Query().Get("action")
	res, err := s.svc.FilterAudit(id, action)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}
