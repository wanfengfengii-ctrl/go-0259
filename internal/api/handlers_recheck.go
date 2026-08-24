package api

import (
	"encoding/json"
	"net/http"

	"mycocycle-growbag-transfer-gate/internal/domain"
	"mycocycle-growbag-transfer-gate/internal/service"
)

// handleContaminationRechecks records contamination recheck evidence.
func (s *Server) handleContaminationRechecks(w http.ResponseWriter, r *http.Request) {
	id := domain.TaskID(r.PathValue("id"))
	var req service.ContaminationRecheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, decodeErr(err))
		return
	}
	res, err := s.svc.ContaminationRecheck(id, req)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// handleReviews submits an independent review.
func (s *Server) handleReviews(w http.ResponseWriter, r *http.Request) {
	id := domain.TaskID(r.PathValue("id"))
	var req service.ReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, decodeErr(err))
		return
	}
	res, err := s.svc.Review(id, req)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// handleFinalize competes to write the single terminal conclusion.
func (s *Server) handleFinalize(w http.ResponseWriter, r *http.Request) {
	id := domain.TaskID(r.PathValue("id"))
	var req service.FinalizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, decodeErr(err))
		return
	}
	res, err := s.svc.Finalize(id, req)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}
