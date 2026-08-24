package service

import (
	"sync"

	"mycocycle-growbag-transfer-gate/internal/domain"
)

// ScriptRegistry maps device invocations to scripted outcomes, letting tests
// and the API inject deterministic device behavior (accept, reject, disconnect,
// timeout, format error). A device with no script always accepts.
type ScriptRegistry struct {
	mu      sync.Mutex
	scripts map[string]map[int]domain.AttemptResult
}

// NewScriptRegistry returns an empty registry.
func NewScriptRegistry() *ScriptRegistry {
	return &ScriptRegistry{scripts: make(map[string]map[int]domain.AttemptResult)}
}

func scriptKey(dt domain.DeviceType, id domain.DeviceID) string {
	return string(dt) + ":" + string(id)
}

// Set records the outcome for a device at a given script sequence number.
func (r *ScriptRegistry) Set(dt domain.DeviceType, id domain.DeviceID, seq int, res domain.AttemptResult) {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := scriptKey(dt, id)
	if r.scripts[k] == nil {
		r.scripts[k] = make(map[int]domain.AttemptResult)
	}
	r.scripts[k][seq] = res
}

// Clear removes all scripts for a device.
func (r *ScriptRegistry) Clear(dt domain.DeviceType, id domain.DeviceID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.scripts, scriptKey(dt, id))
}

// Result returns the scripted outcome for a device invocation, defaulting to
// accepted when no script is registered.
func (r *ScriptRegistry) Result(dt domain.DeviceType, id domain.DeviceID, seq int) domain.AttemptResult {
	r.mu.Lock()
	defer r.mu.Unlock()
	if m, ok := r.scripts[scriptKey(dt, id)]; ok {
		if res, ok := m[seq]; ok {
			delete(m, seq)
			return res
		}
	}
	return domain.AttemptAccepted
}
