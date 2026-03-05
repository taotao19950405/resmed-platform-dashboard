package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	_ "github.com/lib/pq"
	"database/sql"
)

func TestMain(m *testing.M) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn != "" {
		var err error
		db, err = sql.Open("postgres", dsn)
		if err == nil && db.Ping() == nil {
			seed(db)
		}
	}
	os.Exit(m.Run())
}

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
	jsonResponse(rr, 200, map[string]string{"status": "ok"})
	if rr.Code != 200 {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if rr.Header().Get("Content-Type") != "application/json" {
		t.Error("expected application/json content type")
	}
	var result map[string]string
	json.NewDecoder(rr.Body).Decode(&result)
	if result["status"] != "ok" {
		t.Errorf("unexpected body: %v", result)
	}
}

func TestInstrumentMiddleware(t *testing.T) {
	called := false
	handler := instrument(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(200)
	}, "/test")
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()
	handler(rr, req)
	if !called {
		t.Error("inner handler was not called")
	}
}

func TestResponseWriter_DefaultStatus(t *testing.T) {
	rr := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: rr, status: 200}
	if rw.status != 200 {
		t.Errorf("expected default status 200, got %d", rw.status)
	}
}

func TestResponseWriter_WriteHeader(t *testing.T) {
	rr := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: rr, status: 200}
	rw.WriteHeader(404)
	if rw.status != 404 {
		t.Errorf("expected status 404, got %d", rw.status)
	}
}

func TestGetDevice_NoDBReturns500(t *testing.T) {
	db = nil
	req := httptest.NewRequest(http.MethodGet, "/devices/RS-AS11-AU", nil)
	rr := httptest.NewRecorder()
	getDevice(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 with nil db, got %d", rr.Code)
	}
}

func TestJsonResponse_CreatesValidJSON(t *testing.T) {
	rr := httptest.NewRecorder()
	type payload struct {
		SKU  string  `json:"sku"`
		Price float64 `json:"price"`
	}
	jsonResponse(rr, 200, payload{SKU: "RS-AS11-AU", Price: 1299.00})
	var result payload
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Errorf("could not decode response: %v", err)
	}
	if result.SKU != "RS-AS11-AU" {
		t.Errorf("expected SKU RS-AS11-AU, got %s", result.SKU)
	}
}

func TestInstrumentMiddleware_RecordsStatus(t *testing.T) {
	handler := instrument(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}, "/devices")
	req := httptest.NewRequest(http.MethodGet, "/devices/missing", nil)
	rr := httptest.NewRecorder()
	handler(rr, req)
	if rr.Code != 404 {
		t.Errorf("expected 404, got %d", rr.Code)
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

func TestCountDevices_DBNilReturnsZero(t *testing.T) {
	db = nil
	req := httptest.NewRequest(http.MethodGet, "/devices/count", nil)
	rr := httptest.NewRecorder()
	countDevices(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	var result map[string]int
	json.NewDecoder(rr.Body).Decode(&result)
	if result["count"] != 0 {
		t.Errorf("expected count 0 with nil db, got %d", result["count"])
	}
}

func TestDeviceHandler_CountRoute(t *testing.T) {
	db = nil
	req := httptest.NewRequest(http.MethodGet, "/devices/count", nil)
	rr := httptest.NewRecorder()
	deviceHandler(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 from /devices/count, got %d", rr.Code)
	}
}

// ── Integration tests (require DATABASE_URL env) ──────────────────────────

func skipIfNoDB(t *testing.T) {
	t.Helper()
	if db == nil {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}
}

func TestIntegration_HealthHandler_Healthy(t *testing.T) {
	skipIfNoDB(t)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	healthHandler(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	var result map[string]string
	json.NewDecoder(rr.Body).Decode(&result)
	if result["status"] != "healthy" {
		t.Errorf("expected healthy, got %q", result["status"])
	}
}

func TestIntegration_ListDevices(t *testing.T) {
	skipIfNoDB(t)
	req := httptest.NewRequest(http.MethodGet, "/devices", nil)
	rr := httptest.NewRecorder()
	listDevices(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	var devices []Device
	json.NewDecoder(rr.Body).Decode(&devices)
	if len(devices) == 0 {
		t.Error("expected seeded devices, got empty list")
	}
}

func TestIntegration_ListDevices_CategoryFilter(t *testing.T) {
	skipIfNoDB(t)
	req := httptest.NewRequest(http.MethodGet, "/devices?category=mask", nil)
	rr := httptest.NewRecorder()
	listDevices(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	var devices []Device
	json.NewDecoder(rr.Body).Decode(&devices)
	for _, d := range devices {
		if d.Category != "mask" {
			t.Errorf("expected only masks, got category %q", d.Category)
		}
	}
}

func TestIntegration_GetDevice_Found(t *testing.T) {
	skipIfNoDB(t)
	req := httptest.NewRequest(http.MethodGet, "/devices/RS-AS11-AU", nil)
	rr := httptest.NewRecorder()
	getDevice(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	var d Device
	json.NewDecoder(rr.Body).Decode(&d)
	if d.SKU != "RS-AS11-AU" {
		t.Errorf("expected SKU RS-AS11-AU, got %q", d.SKU)
	}
}

func TestIntegration_GetDevice_NotFound(t *testing.T) {
	skipIfNoDB(t)
	req := httptest.NewRequest(http.MethodGet, "/devices/NO-SUCH-SKU", nil)
	rr := httptest.NewRecorder()
	getDevice(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestIntegration_CountDevices_WithDB(t *testing.T) {
	skipIfNoDB(t)
	req := httptest.NewRequest(http.MethodGet, "/devices/count", nil)
	rr := httptest.NewRecorder()
	countDevices(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	var result map[string]int
	json.NewDecoder(rr.Body).Decode(&result)
	if result["count"] < 14 {
		t.Errorf("expected at least 14 seeded devices, got %d", result["count"])
	}
}

func TestIntegration_DeviceHandler_ListRoute(t *testing.T) {
	skipIfNoDB(t)
	req := httptest.NewRequest(http.MethodGet, "/devices", nil)
	rr := httptest.NewRecorder()
	deviceHandler(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 from deviceHandler /devices, got %d", rr.Code)
	}
}

func TestIntegration_DeviceHandler_GetRoute(t *testing.T) {
	skipIfNoDB(t)
	req := httptest.NewRequest(http.MethodGet, "/devices/RS-F40-AU", nil)
	rr := httptest.NewRecorder()
	deviceHandler(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 from deviceHandler /devices/:sku, got %d", rr.Code)
	}
}
