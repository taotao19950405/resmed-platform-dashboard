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
	jsonResponse(rr, 201, map[string]string{"order_id": "1"})
	if rr.Code != 201 {
		t.Errorf("expected 201, got %d", rr.Code)
	}
}

func TestCreateOrder_InvalidBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewBufferString("not-json"))
	rr := httptest.NewRecorder()
	createOrder(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestCreateOrder_ValidBodyNoDB(t *testing.T) {
	db = nil
	body := map[string]any{
		"customer_email":   "test@example.com",
		"shipping_address": "1 Test St",
		"items": []map[string]any{
			{"sku": "RS-AS11-AU", "name": "AirSense 11", "quantity": 1, "unit_price_aud": 1299.00},
		},
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewBuffer(b))
	rr := httptest.NewRecorder()
	createOrder(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 with nil db, got %d", rr.Code)
	}
}

func TestGetOrder_InvalidID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/orders/abc", nil)
	rr := httptest.NewRecorder()
	getOrder(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestResponseWriter_WriteHeader(t *testing.T) {
	rr := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: rr, status: 200}
	rw.WriteHeader(201)
	if rw.status != 201 {
		t.Errorf("expected 201, got %d", rw.status)
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

func TestOrdersHandler_InvalidPostBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewBufferString("invalid"))
	rr := httptest.NewRecorder()
	ordersHandler(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid post body, got %d", rr.Code)
	}
}
