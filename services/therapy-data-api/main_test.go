package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler_DBNil(t *testing.T) {
	db = nil
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	healthHandler(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rr.Code)
	}
}

func TestJsonResponse(t *testing.T) {
	rr := httptest.NewRecorder()
	jsonResponse(rr, 200, map[string]string{"service": "therapy-data-api"})
	if rr.Code != 200 {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if rr.Header().Get("Content-Type") != "application/json" {
		t.Error("expected application/json")
	}
}

func TestGetCompliance_NoDB(t *testing.T) {
	db = nil
	req := httptest.NewRequest(http.MethodGet, "/therapy/compliance/AS11-AU-000142", nil)
	rr := httptest.NewRecorder()
	getCompliance(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 with nil db, got %d", rr.Code)
	}
}

func TestResponseWriter_WriteHeader(t *testing.T) {
	rr := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: rr, status: 200}
	rw.WriteHeader(422)
	if rw.status != 422 {
		t.Errorf("expected 422, got %d", rw.status)
	}
}

func TestInstrumentMiddleware(t *testing.T) {
	reached := false
	h := instrument(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		w.WriteHeader(200)
	}, "/therapy")
	req := httptest.NewRequest(http.MethodGet, "/therapy", nil)
	rr := httptest.NewRecorder()
	h(rr, req)
	if !reached {
		t.Error("inner handler not called")
	}
}

func TestHealthHandler_ReturnsJSON(t *testing.T) {
	db = nil
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	healthHandler(rr, req)
	if rr.Header().Get("Content-Type") != "application/json" {
		t.Error("health must return JSON")
	}
}

func TestInstrumentMiddleware_RecordsStatus(t *testing.T) {
	h := instrument(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
	}, "/therapy")
	req := httptest.NewRequest(http.MethodPost, "/therapy", nil)
	rr := httptest.NewRecorder()
	h(rr, req)
	if rr.Code != 201 {
		t.Errorf("expected 201, got %d", rr.Code)
	}
}

func TestComplianceSummary_Fields(t *testing.T) {
	c := ComplianceSummary{
		SerialNumber:    "AS11-AU-000142",
		AvgUsageHours:   6.5,
		AvgAHI:          2.1,
		CompliantNights: 28,
		TotalNights:     30,
	}
	c.ComplianceRate = float64(c.CompliantNights) / float64(c.TotalNights) * 100
	if c.ComplianceRate < 90 {
		t.Errorf("expected >= 90%% compliance, got %.1f", c.ComplianceRate)
	}
}
