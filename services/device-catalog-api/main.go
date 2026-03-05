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
)

func init() {
	prometheus.MustRegister(httpRequests, httpDuration)
}

type Device struct {
	ID          int     `json:"id"`
	SKU         string  `json:"sku"`
	Name        string  `json:"name"`
	Category    string  `json:"category"`
	PriceAUD    float64 `json:"price_aud"`
	Description string  `json:"description"`
	InStock     bool    `json:"in_stock"`
	CreatedAt   string  `json:"created_at"`
}

func instrument(next http.HandlerFunc, path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: 200}
		next(rw, r)
		dur := time.Since(start).Seconds()
		status := strconv.Itoa(rw.status)
		httpRequests.WithLabelValues(r.Method, path, status).Inc()
		httpDuration.WithLabelValues(r.Method, path).Observe(dur)
	}
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func jsonResponse(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
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
	jsonResponse(w, 200, map[string]string{"status": "healthy", "service": "device-catalog-api"})
}

func listDevices(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	var rows *sql.Rows
	var err error
	if category != "" {
		rows, err = db.Query(`SELECT id, sku, name, category, price_aud, description, in_stock, created_at FROM devices WHERE category=$1 ORDER BY name`, category)
	} else if db != nil {
		rows, err = db.Query(`SELECT id, sku, name, category, price_aud, description, in_stock, created_at FROM devices ORDER BY name`)
	}
	if err != nil {
		jsonResponse(w, 500, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()
	devices := []Device{}
	for rows.Next() {
		var d Device
		rows.Scan(&d.ID, &d.SKU, &d.Name, &d.Category, &d.PriceAUD, &d.Description, &d.InStock, &d.CreatedAt)
		devices = append(devices, d)
	}
	jsonResponse(w, 200, devices)
}

func getDevice(w http.ResponseWriter, r *http.Request) {
	sku := r.URL.Path[len("/devices/"):]
	var d Device
	if db == nil { jsonResponse(w, 500, map[string]string{"error": "db not initialized"}); return }
	err := db.QueryRow(`SELECT id, sku, name, category, price_aud, description, in_stock, created_at FROM devices WHERE sku=$1`, sku).
		Scan(&d.ID, &d.SKU, &d.Name, &d.Category, &d.PriceAUD, &d.Description, &d.InStock, &d.CreatedAt)
	if err == sql.ErrNoRows {
		jsonResponse(w, 404, map[string]string{"error": "device not found"})
		return
	}
	if err != nil {
		jsonResponse(w, 500, map[string]string{"error": err.Error()})
		return
	}
	jsonResponse(w, 200, d)
}

func countDevices(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		jsonResponse(w, 200, map[string]int{"count": 0})
		return
	}
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM devices`).Scan(&count)
	jsonResponse(w, 200, map[string]int{"count": count})
}

func deviceHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/devices" || r.URL.Path == "/devices/" {
		listDevices(w, r)
	} else if r.URL.Path == "/devices/count" {
		countDevices(w, r)
	} else if db != nil {
		getDevice(w, r)
	}
}

func seed(db *sql.DB) {
	db.Exec(`CREATE TABLE IF NOT EXISTS devices (
		id SERIAL PRIMARY KEY,
		sku TEXT UNIQUE NOT NULL,
		name TEXT NOT NULL,
		category TEXT NOT NULL,
		price_aud NUMERIC(10,2) NOT NULL,
		description TEXT,
		in_stock BOOLEAN DEFAULT true,
		created_at TIMESTAMPTZ DEFAULT NOW()
	)`)

	devices := [][]any{
		{"RS-AS11-AU", "AirSense 11 AutoSet", "cpap-machine", 1299.00, "Auto-adjusting CPAP with integrated humidifier and cellular connectivity"},
		{"RS-AS10-AU", "AirSense 10 AutoSet", "cpap-machine", 999.00, "Auto-adjusting CPAP with HumidAir heated humidifier"},
		{"RS-MINI-AU", "AirMini AutoSet", "cpap-machine", 1099.00, "World's smallest CPAP machine, travel-ready with waterless humidification"},
		{"RS-VAUTO-AU", "AirCurve 10 VAuto", "bipap-machine", 1899.00, "Variable pressure BiPAP for complex sleep apnea"},
		{"RS-F40-AU", "AirFit F40 Full Face Mask", "mask", 189.00, "Minimal-contact full face mask with magnetic clips"},
		{"RS-F20-AU", "AirFit F20 Full Face Mask", "mask", 179.00, "Full face mask with InfiniSeal cushion"},
		{"RS-N30-AU", "AirFit N30 Nasal Cradle", "mask", 149.00, "Nasal cradle mask for side sleepers"},
		{"RS-N20-AU", "AirFit N20 Nasal Mask", "mask", 159.00, "Nasal mask with magnetic clips and InfiniSeal cushion"},
		{"RS-P30-AU", "AirFit P30i Pillows Mask", "mask", 169.00, "Under-nose pillows mask with top-of-head tube"},
		{"RS-HH-AU", "HumidAir Heated Humidifier", "accessory", 199.00, "Integrated humidifier for AirSense 10 and AirCurve 10"},
		{"RS-CC-AU", "ClimateLineAir Heated Tube", "accessory", 129.00, "Heated tube for optimal humidity delivery"},
		{"RS-FILTER-AU", "Ultra-Fine Filters Pack (6)", "accessory", 24.95, "6-month supply of ultra-fine CPAP filters"},
		{"RS-MASK-CLEAN-AU", "SoClean 3 CPAP Cleaner", "accessory", 399.00, "Automated CPAP equipment sanitiser using activated oxygen"},
		{"RS-TRAVEL-AU", "Travel Bag for AirMini", "accessory", 69.00, "Protective carry case designed for AirMini CPAP"},
	}

	for _, d := range devices {
		db.Exec(`INSERT INTO devices (sku, name, category, price_aud, description) VALUES ($1,$2,$3,$4,$5) ON CONFLICT (sku) DO NOTHING`,
			d[0], d[1], d[2], d[3], d[4])
	}
}

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://resmed:resmed@localhost:5432/device_catalog?sslmode=disable"
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

	mux := http.NewServeMux()
	mux.HandleFunc("/health", instrument(healthHandler, "/health"))
	mux.HandleFunc("/devices", instrument(deviceHandler, "/devices"))
	mux.HandleFunc("/devices/", instrument(deviceHandler, "/devices/:sku"))
	mux.Handle("/metrics", promhttp.Handler())

	log.Println("device-catalog-api listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
