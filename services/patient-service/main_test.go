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
	jsonResponse(rr, 200, map[string]string{"service": "patient-service"})
	if rr.Code != 200 {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestGetPatient_InvalidID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/patients/abc", nil)
	rr := httptest.NewRecorder()
	getPatient(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestGetPatient_ValidIDNoDB(t *testing.T) {
	db = nil
	req := httptest.NewRequest(http.MethodGet, "/patients/1", nil)
	rr := httptest.NewRecorder()
	getPatient(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 with nil db, got %d", rr.Code)
	}
}

func TestPatientsHandler_NonNumericID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/patients/abc", nil)
	rr := httptest.NewRecorder()
	patientsHandler(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for non-numeric id, got %d", rr.Code)
	}
}

func TestResponseWriter_WriteHeader(t *testing.T) {
	rr := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: rr, status: 200}
	rw.WriteHeader(404)
	if rw.status != 404 {
		t.Errorf("expected 404, got %d", rw.status)
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

func TestInstrumentMiddleware(t *testing.T) {
	called := false
	h := instrument(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}, "/patients")
	req := httptest.NewRequest(http.MethodGet, "/patients", nil)
	rr := httptest.NewRecorder()
	h(rr, req)
	if !called {
		t.Error("handler not called")
	}
}
