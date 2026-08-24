package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"mycocycle-growbag-transfer-gate/internal/domain"
)

// decodeErr wraps a malformed request body as a stable client error.
func decodeErr(err error) error {
	return domain.NewError(domain.CodeReadingOutOfRange, "malformed request body", err.Error())
}

// handleClockAdvance advances the controllable logical clock for deterministic
// tests.
func (s *Server) handleClockAdvance(w http.ResponseWriter, r *http.Request) {
	var req struct {
		By domain.LogicalTime `json:"by"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, decodeErr(err))
		return
	}
	var now domain.LogicalTime
	if req.By <= 0 {
		now = s.svc.Clock().Tick()
	} else {
		now = s.svc.Clock().Set(s.svc.Clock().Now() + req.By)
	}
	writeJSON(w, http.StatusOK, map[string]any{"logical_time": now})
}

// handleDeviceScript injects a scripted device outcome.
func (s *Server) handleDeviceScript(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DeviceType domain.DeviceType    `json:"device_type"`
		DeviceID   domain.DeviceID      `json:"device_id"`
		ScriptSeq  int                  `json:"script_seq"`
		Result     domain.AttemptResult `json:"result"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, decodeErr(err))
		return
	}
	switch req.Result {
	case domain.AttemptAccepted, domain.AttemptRejected, domain.AttemptDisconnected,
		domain.AttemptTimeout, domain.AttemptFormatError:
	default:
		writeError(w, domain.NewError(domain.CodeReadingOutOfRange, "unknown script result", string(req.Result)))
		return
	}
	s.svc.Scripts().Set(req.DeviceType, req.DeviceID, req.ScriptSeq, req.Result)
	writeJSON(w, http.StatusOK, map[string]string{"status": fmt.Sprintf("script set for %s:%s@%d", req.DeviceType, req.DeviceID, req.ScriptSeq)})
}
