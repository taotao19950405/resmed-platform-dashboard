package main

import (
	"bytes"
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

func TestJsonResponse_200(t *testing.T) {
	rr := httptest.NewRecorder()
	jsonResponse(rr, 200, map[string]string{"status": "ok"})
	if rr.Code != 200 {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestAdjustStock_InvalidBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPatch, "/inventory/RS-AS11-AU", bytes.NewBufferString("bad"))
	rr := httptest.NewRecorder()
	adjustStock(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestInventoryHandler_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/inventory/RS-AS11-AU", nil)
	rr := httptest.NewRecorder()
	inventoryHandler(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

func TestResponseWriter_WriteHeader(t *testing.T) {
	rr := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: rr, status: 200}
	rw.WriteHeader(503)
	if rw.status != 503 {
		t.Errorf("expected 503, got %d", rw.status)
	}
}

func TestInstrumentMiddleware(t *testing.T) {
	reached := false
	h := instrument(func(w http.ResponseWriter, r *http.Request) {
		reached = true
	}, "/inventory")
	req := httptest.NewRequest(http.MethodGet, "/inventory", nil)
	rr := httptest.NewRecorder()
	h(rr, req)
	if !reached {
		t.Error("handler was not called")
	}
}

func TestAdjustStock_ValidBodyNoDB(t *testing.T) {
	db = nil
	req := httptest.NewRequest(http.MethodPatch, "/inventory/RS-AS11-AU", bytes.NewBufferString(`{"delta":5}`))
	rr := httptest.NewRecorder()
	adjustStock(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 with nil db, got %d", rr.Code)
	}
}

func TestHealthHandler_ReturnsJSON(t *testing.T) {
	db = nil
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	healthHandler(rr, req)
	if rr.Header().Get("Content-Type") != "application/json" {
		t.Error("health endpoint must return JSON")
	}
}
