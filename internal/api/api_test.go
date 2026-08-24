package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"mycocycle-growbag-transfer-gate/internal/catalog"
	"mycocycle-growbag-transfer-gate/internal/domain"
	"mycocycle-growbag-transfer-gate/internal/service"
	"mycocycle-growbag-transfer-gate/internal/store"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	st, err := store.OpenMemory()
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	cat, err := st.LoadCatalog()
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}
	return New(service.New(st, cat, domain.NewClock()))
}

func TestHealthz(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "ok" {
		t.Fatalf("status = %q, want ok", body["status"])
	}
}

func TestLockEndpoint(t *testing.T) {
	srv := newTestServer(t)
	req := catalog.LockRequest{
		StrainRevision:    "strain-po-2024.03",
		SubstrateRevision: "sub-hw-01",
		SubstrateSummary:  "hardwood sawdust + wheat bran 78:20",
		SterilizerSummary: "run-A-2026-08-21",
		InoculationLine:   "line-1",
		BagBatch:          "B-001",
		BagPositions:      []domain.BagPosition{"P1", "P2"},
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
	body, _ := json.Marshal(req)
	hreq := httptest.NewRequest(http.MethodPost, "/v1/tasks/lock", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, hreq)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201: %s", rec.Code, rec.Body.String())
	}
	var res map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if res["task_id"] == "" {
		t.Fatalf("missing task_id in %v", res)
	}
}
