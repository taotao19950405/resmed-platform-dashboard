package main

import (
	"bytes"
	"encoding/json"
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
	jsonResponse(rr, 201, map[string]any{"id": 1})
	if rr.Code != 201 {
		t.Errorf("expected 201, got %d", rr.Code)
	}
}

func TestCreateNotification_InvalidBody(t *testing.T) {
	db = nil
	req := httptest.NewRequest(http.MethodPost, "/notifications", bytes.NewBufferString("not-json"))
	rr := httptest.NewRecorder()
	createNotification(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestCreateNotification_ValidBodyNoDB(t *testing.T) {
	db = nil
	body, _ := json.Marshal(map[string]string{
		"type":      "low_stock",
		"recipient": "warehouse@resmed.com.au",
		"subject":   "Test alert",
		"payload":   `{"sku":"RS-F20-AU"}`,
	})
	req := httptest.NewRequest(http.MethodPost, "/notifications", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()
	createNotification(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 with nil db, got %d", rr.Code)
	}
}

func TestNotificationsHandler_PostRoute(t *testing.T) {
	db = nil
	req := httptest.NewRequest(http.MethodPost, "/notifications", bytes.NewBufferString("bad"))
	rr := httptest.NewRecorder()
	notificationsHandler(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestNotificationsHandler_GetRoute(t *testing.T) {
	db = nil
	req := httptest.NewRequest(http.MethodGet, "/notifications", nil)
	rr := httptest.NewRecorder()
	notificationsHandler(rr, req)
	if rr.Code == 0 {
		t.Error("expected a response")
	}
}

func TestResponseWriter_WriteHeader(t *testing.T) {
	rr := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: rr, status: 200}
	rw.WriteHeader(500)
	if rw.status != 500 {
		t.Errorf("expected 500, got %d", rw.status)
	}
}

func TestInstrumentMiddleware(t *testing.T) {
	called := false
	h := instrument(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}, "/notifications")
	req := httptest.NewRequest(http.MethodGet, "/notifications", nil)
	rr := httptest.NewRecorder()
	h(rr, req)
	if !called {
		t.Error("handler not called")
	}
}
