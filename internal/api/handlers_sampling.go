package api

import (
	"encoding/json"
	"net/http"

	"mycocycle-growbag-transfer-gate/internal/domain"
	"mycocycle-growbag-transfer-gate/internal/service"
)

// handleSamplingConfirmations submits one person's sampling confirmation.
func (s *Server) handleSamplingConfirmations(w http.ResponseWriter, r *http.Request) {
	id := domain.TaskID(r.PathValue("id"))
	var req service.SamplingConfirmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, decodeErr(err))
		return
	}
	res, err := s.svc.SamplingConfirm(id, req)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// handleSampleSeals completes bag-position sample sealing.
func (s *Server) handleSampleSeals(w http.ResponseWriter, r *http.Request) {
	id := domain.TaskID(r.PathValue("id"))
	var req service.SampleSealRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, decodeErr(err))
		return
	}
	res, err := s.svc.SampleSeal(id, req)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}
