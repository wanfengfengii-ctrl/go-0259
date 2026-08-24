package api

import (
	"encoding/json"
	"net/http"

	"mycocycle-growbag-transfer-gate/internal/domain"
	"mycocycle-growbag-transfer-gate/internal/service"
)

// handleObservations submits a day-age x bag-position coverage cell.
func (s *Server) handleObservations(w http.ResponseWriter, r *http.Request) {
	id := domain.TaskID(r.PathValue("id"))
	var req service.ObservationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, decodeErr(err))
		return
	}
	res, err := s.svc.Observation(id, req)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// handleDeviceReadings submits a device reading.
func (s *Server) handleDeviceReadings(w http.ResponseWriter, r *http.Request) {
	id := domain.TaskID(r.PathValue("id"))
	var req service.DeviceReadingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, decodeErr(err))
		return
	}
	res, err := s.svc.DeviceReading(id, req)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}
