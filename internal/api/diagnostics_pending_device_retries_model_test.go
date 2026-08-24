package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"mycocycle-growbag-transfer-gate/internal/api"
	"mycocycle-growbag-transfer-gate/internal/catalog"
	"mycocycle-growbag-transfer-gate/internal/domain"
	"mycocycle-growbag-transfer-gate/internal/service"
	"mycocycle-growbag-transfer-gate/internal/store"
)

func TestModel_DiagnosticsPendingDeviceRetriesAreScopedByTaskAndDevice(t *testing.T) {
	openServer := func(t *testing.T, dbPath string) (*api.Server, *store.SQLite) {
		t.Helper()
		st, err := store.Open(dbPath)
		if err != nil {
			t.Fatalf("open store: %v", err)
		}
		cat, err := st.LoadCatalog()
		if err != nil {
			t.Fatalf("load catalog: %v", err)
		}
		return api.New(service.New(st, cat, domain.NewClock()), api.WithRequestLogging(false)), st
	}
	post := func(t *testing.T, srv http.Handler, path string, body any, wantStatus int, out any) {
		t.Helper()
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal %s: %v", path, err)
		}
		req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)
		if rec.Code != wantStatus {
			t.Fatalf("POST %s status = %d, want %d: %s", path, rec.Code, wantStatus, rec.Body.String())
		}
		if out != nil {
			if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
				t.Fatalf("decode %s: %v", path, err)
			}
		}
	}
	get := func(t *testing.T, srv http.Handler, path string, out any) {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d, want 200: %s", path, rec.Code, rec.Body.String())
		}
		if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
			t.Fatalf("decode %s: %v", path, err)
		}
	}
	lockRequest := func(batch domain.BagBatch, positions []domain.BagPosition, rack domain.RackID) catalog.LockRequest {
		return catalog.LockRequest{
			StrainRevision:    "strain-po-2024.03",
			SubstrateRevision: "sub-hw-01",
			SubstrateSummary:  "hardwood sawdust + wheat bran 78:20",
			SterilizerSummary: "run-A-2026-08-21",
			InoculationLine:   "line-1",
			BagBatch:          batch,
			BagPositions:      positions,
			RackID:            rack,
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
	startTask := func(t *testing.T, srv http.Handler, batch domain.BagBatch, positions []domain.BagPosition, rack domain.RackID, start, end domain.LogicalTime) domain.TaskID {
		t.Helper()
		var locked service.LockResult
		post(t, srv, "/v1/tasks/lock", lockRequest(batch, positions, rack), http.StatusCreated, &locked)
		post(t, srv, "/v1/tasks/"+string(locked.TaskID)+"/sampling-confirmations", service.SamplingConfirmRequest{
			Operation:  "confirm-" + domain.OperationID(batch) + "-alice",
			Generation: 1,
			Person:     "alice",
			Batch:      batch,
			Positions:  positions,
		}, http.StatusOK, nil)
		post(t, srv, "/v1/tasks/"+string(locked.TaskID)+"/sampling-confirmations", service.SamplingConfirmRequest{
			Operation:  "confirm-" + domain.OperationID(batch) + "-bob",
			Generation: 1,
			Person:     "bob",
			Batch:      batch,
			Positions:  positions,
		}, http.StatusOK, nil)
		post(t, srv, "/v1/tasks/"+string(locked.TaskID)+"/sample-seals", service.SampleSealRequest{
			Operation:  "seal-" + domain.OperationID(batch),
			Generation: 1,
			Person:     "alice",
			Positions:  positions,
		}, http.StatusOK, nil)
		post(t, srv, "/v1/tasks/"+string(locked.TaskID)+"/occupancy/start", service.OccupancyStartRequest{
			Operation:  "occupy-" + domain.OperationID(batch),
			Generation: 1,
			RackID:     rack,
			ProbeID:    "probe-1",
			ProbeStart: start,
			ProbeEnd:   end,
		}, http.StatusOK, nil)
		return locked.TaskID
	}
	setScript := func(t *testing.T, srv http.Handler, seq int, result domain.AttemptResult) {
		t.Helper()
		post(t, srv, "/v1/devices/scripts", struct {
			DeviceType domain.DeviceType    `json:"device_type"`
			DeviceID   domain.DeviceID      `json:"device_id"`
			ScriptSeq  int                  `json:"script_seq"`
			Result     domain.AttemptResult `json:"result"`
		}{
			DeviceType: domain.DeviceProbe,
			DeviceID:   "probe-1",
			ScriptSeq:  seq,
			Result:     result,
		}, http.StatusOK, nil)
	}
	readProbe := func(t *testing.T, srv http.Handler, taskID domain.TaskID, op domain.OperationID, seq int, position domain.BagPosition) service.DeviceReadingResult {
		t.Helper()
		var out service.DeviceReadingResult
		post(t, srv, "/v1/tasks/"+string(taskID)+"/device-readings", service.DeviceReadingRequest{
			Operation:  op,
			Generation: 1,
			DeviceType: domain.DeviceProbe,
			DeviceID:   "probe-1",
			Metric:     domain.MetricTemperature,
			Value:      "25.0",
			Position:   position,
			DayAge:     1,
			ScriptSeq:  seq,
		}, http.StatusOK, &out)
		return out
	}

	cases := []struct {
		name         string
		arrange      func(*testing.T, http.Handler) []domain.TaskID
		wantPending  int
		wantAttempts int
		wantReadings []int
	}{
		{
			name: "same task accepted retry closes failed probe attempt",
			arrange: func(t *testing.T, srv http.Handler) []domain.TaskID {
				taskID := startTask(t, srv, "B-001", []domain.BagPosition{"P1", "P2"}, "R1", 1, 100)
				setScript(t, srv, 1, domain.AttemptDisconnected)
				if got := readProbe(t, srv, taskID, "read-disconnect", 1, "P1"); got.Result != domain.AttemptDisconnected || got.RetryCount != 1 {
					t.Fatalf("first attempt = %s/%d, want disconnected/1", got.Result, got.RetryCount)
				}
				if got := readProbe(t, srv, taskID, "read-accepted", 2, "P1"); got.Result != domain.AttemptAccepted || got.RetryCount != 2 {
					t.Fatalf("retry attempt = %s/%d, want accepted/2", got.Result, got.RetryCount)
				}
				return []domain.TaskID{taskID}
			},
			wantPending:  0,
			wantAttempts: 2,
			wantReadings: []int{1},
		},
		{
			name: "later task accepted read does not close earlier task failure on same probe",
			arrange: func(t *testing.T, srv http.Handler) []domain.TaskID {
				first := startTask(t, srv, "B-001", []domain.BagPosition{"P1", "P2"}, "R1", 1, 50)
				setScript(t, srv, 1, domain.AttemptDisconnected)
				if got := readProbe(t, srv, first, "first-disconnect", 1, "P1"); got.Result != domain.AttemptDisconnected || got.RetryCount != 1 {
					t.Fatalf("first task attempt = %s/%d, want disconnected/1", got.Result, got.RetryCount)
				}
				second := startTask(t, srv, "B-002", []domain.BagPosition{"P3", "P4"}, "R2", 50, 100)
				if got := readProbe(t, srv, second, "second-accepted", 2, "P3"); got.Result != domain.AttemptAccepted || got.RetryCount != 1 {
					t.Fatalf("second task attempt = %s/%d, want accepted/1", got.Result, got.RetryCount)
				}
				return []domain.TaskID{first, second}
			},
			wantPending:  1,
			wantAttempts: 2,
			wantReadings: []int{0, 1},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dbPath := filepath.Join(t.TempDir(), "mycocycle.db")
			srv, st := openServer(t, dbPath)
			taskIDs := tc.arrange(t, srv)
			if err := st.Close(); err != nil {
				t.Fatalf("close store before restart: %v", err)
			}

			srv, st = openServer(t, dbPath)
			defer st.Close()

			var report service.Report
			get(t, srv, "/v1/diagnostics", &report)
			if report.PendingRetries != tc.wantPending {
				t.Fatalf("diagnostics pending_device_retries = %d, want %d", report.PendingRetries, tc.wantPending)
			}
			if report.Recovery.PendingDeviceRetries != tc.wantPending {
				t.Fatalf("recovery pending_device_retries = %d, want %d", report.Recovery.PendingDeviceRetries, tc.wantPending)
			}
			if report.Recovery.DeviceAttempts != tc.wantAttempts {
				t.Fatalf("device_attempts = %d, want %d", report.Recovery.DeviceAttempts, tc.wantAttempts)
			}
			if len(report.Tasks) != len(tc.wantReadings) {
				t.Fatalf("diagnostics tasks = %d, want %d", len(report.Tasks), len(tc.wantReadings))
			}

			for i, taskID := range taskIDs {
				var detail service.TaskDetail
				get(t, srv, "/v1/tasks/"+string(taskID)+"/detail", &detail)
				if len(detail.Readings) != tc.wantReadings[i] {
					t.Fatalf("task %s readings = %d, want %d", taskID, len(detail.Readings), tc.wantReadings[i])
				}
			}
		})
	}
}
