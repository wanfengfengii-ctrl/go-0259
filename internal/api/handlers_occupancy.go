package api

import (
	"encoding/json"
	"net/http"

	"mycocycle-growbag-transfer-gate/internal/domain"
	"mycocycle-growbag-transfer-gate/internal/service"
)

// handleOccupancyStart starts rack and probe window occupancy.
func (s *Server) handleOccupancyStart(w http.ResponseWriter, r *http.Request) {
	id := domain.TaskID(r.PathValue("id"))
	var req service.OccupancyStartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, decodeErr(err))
		return
	}
	res, err := s.svc.OccupancyStart(id, req)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// handleOccupancyMoveRack moves a task to a different rack and probe window.
func (s *Server) handleOccupancyMoveRack(w http.ResponseWriter, r *http.Request) {
	id := domain.TaskID(r.PathValue("id"))
	var req service.OccupancyMoveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, decodeErr(err))
		return
	}
	res, err := s.svc.OccupancyMoveRack(id, req)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}
