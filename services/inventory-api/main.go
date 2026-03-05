package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var db *sql.DB

var (
	httpRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total HTTP requests",
	}, []string{"method", "path", "status"})

	httpDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request latency",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path"})

	lowStockItems = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "inventory_low_stock_items_total",
		Help: "Number of SKUs currently below reorder threshold",
	})
)

func init() {
	prometheus.MustRegister(httpRequests, httpDuration, lowStockItems)
}

type InventoryItem struct {
	ID                int     `json:"id"`
	SKU               string  `json:"sku"`
	Name              string  `json:"name"`
	Quantity          int     `json:"quantity"`
	ReorderThreshold  int     `json:"reorder_threshold"`
	WarehouseLocation string  `json:"warehouse_location"`
	UnitCostAUD       float64 `json:"unit_cost_aud"`
	LowStock          bool    `json:"low_stock"`
	UpdatedAt         string  `json:"updated_at"`
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func instrument(next http.HandlerFunc, path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: 200}
		next(rw, r)
		httpRequests.WithLabelValues(r.Method, path, strconv.Itoa(rw.status)).Inc()
		httpDuration.WithLabelValues(r.Method, path).Observe(time.Since(start).Seconds())
	}
}

func jsonResponse(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func updateLowStockGauge() {
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM inventory WHERE quantity <= reorder_threshold`).Scan(&count)
	lowStockItems.Set(float64(count))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		jsonResponse(w, 503, map[string]string{"status": "unhealthy", "error": "db not initialized"})
		return
	}
	if err := db.Ping(); err != nil {
		jsonResponse(w, 503, map[string]string{"status": "unhealthy", "error": err.Error()})
		return
	}
	jsonResponse(w, 200, map[string]string{"status": "healthy", "service": "inventory-api"})
}

func listInventory(w http.ResponseWriter, r *http.Request) {
	lowStock := r.URL.Query().Get("low_stock")
	var rows *sql.Rows
	var err error
	if lowStock == "true" {
		rows, err = db.Query(`SELECT id, sku, name, quantity, reorder_threshold, warehouse_location, unit_cost_aud, updated_at FROM inventory WHERE quantity <= reorder_threshold ORDER BY quantity ASC`)
	} else {
		rows, err = db.Query(`SELECT id, sku, name, quantity, reorder_threshold, warehouse_location, unit_cost_aud, updated_at FROM inventory ORDER BY name`)
	}
	if err != nil {
		jsonResponse(w, 500, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()
	items := []InventoryItem{}
	for rows.Next() {
		var it InventoryItem
		rows.Scan(&it.ID, &it.SKU, &it.Name, &it.Quantity, &it.ReorderThreshold, &it.WarehouseLocation, &it.UnitCostAUD, &it.UpdatedAt)
		it.LowStock = it.Quantity <= it.ReorderThreshold
		items = append(items, it)
	}
	jsonResponse(w, 200, items)
}

func adjustStock(w http.ResponseWriter, r *http.Request) {
	sku := r.URL.Path[len("/inventory/"):]
	var req struct {
		Delta int    `json:"delta"`
		Note  string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, 400, map[string]string{"error": "invalid body"})
		return
	}
	if db == nil { jsonResponse(w, 500, map[string]string{"error": "db not initialized"}); return }
	result, err := db.Exec(`UPDATE inventory SET quantity = quantity + $1, updated_at = NOW() WHERE sku = $2`, req.Delta, sku)
	if err != nil {
		jsonResponse(w, 500, map[string]string{"error": err.Error()})
		return
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		jsonResponse(w, 404, map[string]string{"error": "SKU not found"})
		return
	}
	updateLowStockGauge()
	jsonResponse(w, 200, map[string]string{"status": "updated", "sku": sku})
}

func inventoryHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/inventory" || r.URL.Path == "/inventory/" {
		listInventory(w, r)
	} else if r.Method == http.MethodPatch {
		adjustStock(w, r)
	} else {
		jsonResponse(w, 405, map[string]string{"error": "method not allowed"})
	}
}

func seed(db *sql.DB) {
	db.Exec(`CREATE TABLE IF NOT EXISTS inventory (
		id SERIAL PRIMARY KEY,
		sku TEXT UNIQUE NOT NULL,
		name TEXT NOT NULL,
		quantity INT NOT NULL DEFAULT 0,
		reorder_threshold INT NOT NULL DEFAULT 10,
		warehouse_location TEXT,
		unit_cost_aud NUMERIC(10,2),
		updated_at TIMESTAMPTZ DEFAULT NOW()
	)`)

	items := [][]any{
		{"RS-AS11-AU", "AirSense 11 AutoSet", 42, 10, "SYD-A1-01", 720.00},
		{"RS-AS10-AU", "AirSense 10 AutoSet", 87, 15, "SYD-A1-02", 550.00},
		{"RS-MINI-AU", "AirMini AutoSet", 23, 10, "SYD-A2-01", 610.00},
		{"RS-VAUTO-AU", "AirCurve 10 VAuto", 8, 10, "SYD-A2-02", 1050.00},
		{"RS-F40-AU", "AirFit F40 Full Face Mask", 156, 30, "SYD-B1-01", 95.00},
		{"RS-F20-AU", "AirFit F20 Full Face Mask", 7, 30, "SYD-B1-02", 88.00},
		{"RS-N30-AU", "AirFit N30 Nasal Cradle", 203, 30, "SYD-B2-01", 72.00},
		{"RS-N20-AU", "AirFit N20 Nasal Mask", 91, 30, "SYD-B2-02", 80.00},
		{"RS-P30-AU", "AirFit P30i Pillows Mask", 4, 30, "SYD-B3-01", 84.00},
		{"RS-HH-AU", "HumidAir Heated Humidifier", 65, 20, "SYD-C1-01", 110.00},
		{"RS-CC-AU", "ClimateLineAir Heated Tube", 112, 20, "SYD-C1-02", 65.00},
		{"RS-FILTER-AU", "Ultra-Fine Filters Pack (6)", 340, 50, "SYD-C2-01", 12.00},
		{"RS-MASK-CLEAN-AU", "SoClean 3 CPAP Cleaner", 6, 10, "SYD-C2-02", 220.00},
		{"RS-TRAVEL-AU", "Travel Bag for AirMini", 38, 15, "SYD-C3-01", 35.00},
	}
	for _, it := range items {
		db.Exec(`INSERT INTO inventory (sku, name, quantity, reorder_threshold, warehouse_location, unit_cost_aud) VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT (sku) DO NOTHING`,
			it[0], it[1], it[2], it[3], it[4], it[5])
	}
	updateLowStockGauge()
}

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://resmed:resmed@localhost:5432/inventory?sslmode=disable"
	}
	var err error
	for i := 0; i < 10; i++ {
		db, err = sql.Open("postgres", dsn)
		if err == nil {
			if err = db.Ping(); err == nil {
				break
			}
		}
		log.Printf("waiting for postgres... (%d/10)", i+1)
		time.Sleep(3 * time.Second)
	}
	if err != nil {
		log.Fatalf("cannot connect to postgres: %v", err)
	}
	seed(db)

	go func() {
		for range time.Tick(30 * time.Second) {
			updateLowStockGauge()
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", instrument(healthHandler, "/health"))
	mux.HandleFunc("/inventory", instrument(inventoryHandler, "/inventory"))
	mux.HandleFunc("/inventory/", instrument(inventoryHandler, "/inventory/:sku"))
	mux.Handle("/metrics", promhttp.Handler())

	log.Println("inventory-api listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
