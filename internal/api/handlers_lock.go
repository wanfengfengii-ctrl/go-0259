package api

import (
	"encoding/json"
	"net/http"

	"mycocycle-growbag-transfer-gate/internal/catalog"
)

// handleLock creates and locks a transfer inspection task.
func (s *Server) handleLock(w http.ResponseWriter, r *http.Request) {
	var req catalog.LockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, decodeErr(err))
		return
	}
	res, err := s.svc.Lock(req)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, res)
}
