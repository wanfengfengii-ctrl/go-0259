package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"mycocycle-growbag-transfer-gate/internal/catalog"
	"mycocycle-growbag-transfer-gate/internal/domain"
)

func TestModel_LockConflictRollsBackFailedBatchState(t *testing.T) {
	type taskSummary struct {
		TaskID     string `json:"task_id"`
		BagBatch   string `json:"bag_batch"`
		Generation int    `json:"generation"`
		State      string `json:"state"`
	}

	cases := []struct {
		name                 string
		firstBatch           domain.BagBatch
		firstPositions       []domain.BagPosition
		conflictingBatch     domain.BagBatch
		conflictingPositions []domain.BagPosition
	}{
		{
			name:                 "overlapping bag position conflict leaves only the successful batch state",
			firstBatch:           "B-001",
			firstPositions:       []domain.BagPosition{"P1", "P2"},
			conflictingBatch:     "B-002",
			conflictingPositions: []domain.BagPosition{"P1"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := newTestServer(t)
			lockRequest := func(batch domain.BagBatch, positions []domain.BagPosition) catalog.LockRequest {
				return catalog.LockRequest{
					StrainRevision:    "strain-po-2024.03",
					SubstrateRevision: "sub-hw-01",
					SubstrateSummary:  "hardwood sawdust + wheat bran 78:20",
					SterilizerSummary: "run-A-2026-08-21",
					InoculationLine:   "line-1",
					BagBatch:          batch,
					BagPositions:      append([]domain.BagPosition(nil), positions...),
					RackID:            "R1",
					ProbeWindow:       catalog.ProbeWindow{ProbeID: "probe-1", Start: 1, End: 100},
					Schedule:          catalog.ScheduleTemplate{DayAges: []domain.DayAge{1, 2}},
					Thresholds: catalog.Thresholds{
						Contamination: domain.MustFixed(0, 0),
						MaturityMin:   domain.MustFixed(700, 1),
						MaturityMax:   domain.MustFixed(1000, 1),
						MoistureMin:   domain.MustFixed(600, 1),
						MoistureMax:   domain.MustFixed(700, 1),
						PHMin:         domain.MustFixed(550, 2),
						PHMax:         domain.MustFixed(650, 2),
					},
					Reviewers: []domain.PersonID{"alice", "bob", "carol", "dave"},
				}
			}
			postLock := func(req catalog.LockRequest) (int, []byte) {
				t.Helper()
				body, err := json.Marshal(req)
				if err != nil {
					t.Fatalf("marshal lock request: %v", err)
				}
				hreq := httptest.NewRequest(http.MethodPost, "/v1/tasks/lock", bytes.NewReader(body))
				rec := httptest.NewRecorder()
				srv.ServeHTTP(rec, hreq)
				return rec.Code, rec.Body.Bytes()
			}
			getJSON := func(path string, out any) {
				t.Helper()
				hreq := httptest.NewRequest(http.MethodGet, path, nil)
				rec := httptest.NewRecorder()
				srv.ServeHTTP(rec, hreq)
				if rec.Code != http.StatusOK {
					t.Fatalf("GET %s status = %d, want 200: %s", path, rec.Code, rec.Body.String())
				}
				if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
					t.Fatalf("decode GET %s response: %v", path, err)
				}
			}

			status, body := postLock(lockRequest(tc.firstBatch, tc.firstPositions))
			if status != http.StatusCreated {
				t.Fatalf("first lock status = %d, want 201: %s", status, string(body))
			}
			var first struct {
				TaskID string `json:"task_id"`
				State  string `json:"state"`
			}
			if err := json.Unmarshal(body, &first); err != nil {
				t.Fatalf("decode first lock response: %v", err)
			}
			if first.TaskID == "" {
				t.Fatal("first lock returned an empty task_id")
			}
			if first.State != "pending_sampling" {
				t.Fatalf("first lock state = %q, want pending_sampling", first.State)
			}

			status, body = postLock(lockRequest(tc.conflictingBatch, tc.conflictingPositions))
			if status != http.StatusConflict {
				t.Fatalf("conflicting lock status = %d, want 409: %s", status, string(body))
			}
			var conflict struct {
				Code domain.ErrorCode `json:"code"`
			}
			if err := json.Unmarshal(body, &conflict); err != nil {
				t.Fatalf("decode conflict response: %v", err)
			}
			if conflict.Code != domain.CodeResourceWindowConflict {
				t.Fatalf("conflict code = %q, want %q", conflict.Code, domain.CodeResourceWindowConflict)
			}

			var failedBatchTasks []taskSummary
			getJSON("/v1/tasks?batch="+string(tc.conflictingBatch), &failedBatchTasks)
			if len(failedBatchTasks) != 0 {
				t.Fatalf("conflicting batch tasks = %+v, want none", failedBatchTasks)
			}

			var firstBatchTasks []taskSummary
			getJSON("/v1/tasks?batch="+string(tc.firstBatch), &firstBatchTasks)
			if len(firstBatchTasks) != 1 {
				t.Fatalf("first batch tasks = %+v, want exactly the successful task", firstBatchTasks)
			}
			if firstBatchTasks[0].TaskID != first.TaskID || firstBatchTasks[0].State != "pending_sampling" {
				t.Fatalf("first batch task = %+v, want task_id %s in pending_sampling", firstBatchTasks[0], first.TaskID)
			}

			var detail struct {
				BagPositions []struct {
					TaskID   string `json:"task_id"`
					Batch    string `json:"batch"`
					Position string `json:"position"`
				} `json:"bag_positions"`
				Leases []struct {
					ResourceType string `json:"resource_type"`
					ResourceID   string `json:"resource_id"`
					TaskID       string `json:"task_id"`
					State        string `json:"state"`
				} `json:"leases"`
			}
			getJSON("/v1/tasks/"+first.TaskID+"/detail", &detail)
			if len(detail.BagPositions) != len(tc.firstPositions) {
				t.Fatalf("successful task locked positions = %+v, want %d", detail.BagPositions, len(tc.firstPositions))
			}
			for _, p := range detail.BagPositions {
				if p.TaskID != first.TaskID || p.Batch != string(tc.firstBatch) {
					t.Fatalf("locked position = %+v, want task_id %s and batch %s", p, first.TaskID, tc.firstBatch)
				}
			}
			if len(detail.Leases) != len(tc.firstPositions)+1 {
				t.Fatalf("successful task leases = %+v, want %d", detail.Leases, len(tc.firstPositions)+1)
			}
			activeResources := make(map[string]bool, len(detail.Leases))
			for _, lease := range detail.Leases {
				if lease.TaskID != first.TaskID || lease.State != "active" {
					t.Fatalf("lease = %+v, want active lease for task_id %s", lease, first.TaskID)
				}
				activeResources[lease.ResourceType+":"+lease.ResourceID] = true
			}
			if !activeResources["bag_batch:"+string(tc.firstBatch)] {
				t.Fatalf("active leases = %+v, missing successful batch lease", detail.Leases)
			}
			for _, position := range tc.firstPositions {
				if !activeResources["bag_position:"+string(position)] {
					t.Fatalf("active leases = %+v, missing position lease %s", detail.Leases, position)
				}
			}

			var diagnostics struct {
				Recovery struct {
					Tasks              int `json:"tasks"`
					ActiveLeases       int `json:"active_leases"`
					LockedBagPositions int `json:"locked_bag_positions"`
				} `json:"recovery"`
				TasksByState map[string]int `json:"tasks_by_state"`
				Tasks        []taskSummary  `json:"tasks"`
			}
			getJSON("/v1/diagnostics", &diagnostics)
			if diagnostics.Recovery.Tasks != 1 {
				t.Fatalf("recovered task count = %d, want 1", diagnostics.Recovery.Tasks)
			}
			if diagnostics.Recovery.LockedBagPositions != len(tc.firstPositions) {
				t.Fatalf("recovered locked position count = %d, want %d", diagnostics.Recovery.LockedBagPositions, len(tc.firstPositions))
			}
			if diagnostics.Recovery.ActiveLeases != len(tc.firstPositions)+1 {
				t.Fatalf("recovered active lease count = %d, want %d", diagnostics.Recovery.ActiveLeases, len(tc.firstPositions)+1)
			}
			if diagnostics.TasksByState["pending_sampling"] != 1 {
				t.Fatalf("tasks_by_state[pending_sampling] = %d, want 1", diagnostics.TasksByState["pending_sampling"])
			}
			if len(diagnostics.Tasks) != 1 || diagnostics.Tasks[0].TaskID != first.TaskID {
				t.Fatalf("diagnostics tasks = %+v, want only successful task %s", diagnostics.Tasks, first.TaskID)
			}
		})
	}
}
