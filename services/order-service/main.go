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

	ordersCreated = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "orders_created_total",
		Help: "Total orders placed",
	})
)

func init() {
	prometheus.MustRegister(httpRequests, httpDuration, ordersCreated)
}

type OrderItem struct {
	ID       int     `json:"id"`
	OrderID  int     `json:"order_id"`
	SKU      string  `json:"sku"`
	Name     string  `json:"name"`
	Qty      int     `json:"quantity"`
	UnitPrice float64 `json:"unit_price_aud"`
}

type Order struct {
	ID              int         `json:"id"`
	CustomerEmail   string      `json:"customer_email"`
	Status          string      `json:"status"`
	TotalAUD        float64     `json:"total_aud"`
	ShippingAddress string      `json:"shipping_address"`
	Items           []OrderItem `json:"items,omitempty"`
	CreatedAt       string      `json:"created_at"`
	UpdatedAt       string      `json:"updated_at"`
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
		dur := time.Since(start).Seconds()
		httpRequests.WithLabelValues(r.Method, path, strconv.Itoa(rw.status)).Inc()
		httpDuration.WithLabelValues(r.Method, path).Observe(dur)
	}
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
	jsonResponse(w, 200, map[string]string{"status": "healthy", "service": "order-service"})
}

func listOrders(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	var rows *sql.Rows
	var err error
	if status != "" {
		rows, err = db.Query(`SELECT id, customer_email, status, total_aud, shipping_address, created_at, updated_at FROM orders WHERE status=$1 ORDER BY created_at DESC LIMIT 100`, status)
	} else if db != nil {
		rows, err = db.Query(`SELECT id, customer_email, status, total_aud, shipping_address, created_at, updated_at FROM orders ORDER BY created_at DESC LIMIT 100`)
	}
	if err != nil {
		jsonResponse(w, 500, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()
	orders := []Order{}
	for rows.Next() {
		var o Order
		rows.Scan(&o.ID, &o.CustomerEmail, &o.Status, &o.TotalAUD, &o.ShippingAddress, &o.CreatedAt, &o.UpdatedAt)
		orders = append(orders, o)
	}
	jsonResponse(w, 200, orders)
}

func getOrder(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/orders/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		jsonResponse(w, 400, map[string]string{"error": "invalid order id"})
		return
	}
	var o Order
	err = db.QueryRow(`SELECT id, customer_email, status, total_aud, shipping_address, created_at, updated_at FROM orders WHERE id=$1`, id).
		Scan(&o.ID, &o.CustomerEmail, &o.Status, &o.TotalAUD, &o.ShippingAddress, &o.CreatedAt, &o.UpdatedAt)
	if err == sql.ErrNoRows {
		jsonResponse(w, 404, map[string]string{"error": "order not found"})
		return
	}
	if err != nil {
		jsonResponse(w, 500, map[string]string{"error": err.Error()})
		return
	}
	rows, _ := db.Query(`SELECT id, order_id, sku, name, quantity, unit_price_aud FROM order_items WHERE order_id=$1`, o.ID)
	defer rows.Close()
	for rows.Next() {
		var item OrderItem
		rows.Scan(&item.ID, &item.OrderID, &item.SKU, &item.Name, &item.Qty, &item.UnitPrice)
		o.Items = append(o.Items, item)
	}
	jsonResponse(w, 200, o)
}

func createOrder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CustomerEmail   string `json:"customer_email"`
		ShippingAddress string `json:"shipping_address"`
		Items           []struct {
			SKU       string  `json:"sku"`
			Name      string  `json:"name"`
			Qty       int     `json:"quantity"`
			UnitPrice float64 `json:"unit_price_aud"`
		} `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, 400, map[string]string{"error": "invalid body"})
		return
	}
	var total float64
	for _, it := range req.Items {
		total += float64(it.Qty) * it.UnitPrice
	}
	var orderID int
	if db == nil { jsonResponse(w, 500, map[string]string{"error": "db not initialized"}); return }
	err := db.QueryRow(`INSERT INTO orders (customer_email, status, total_aud, shipping_address) VALUES ($1,'pending',$2,$3) RETURNING id`,
		req.CustomerEmail, total, req.ShippingAddress).Scan(&orderID)
	if err != nil {
		jsonResponse(w, 500, map[string]string{"error": err.Error()})
		return
	}
	for _, it := range req.Items {
		db.Exec(`INSERT INTO order_items (order_id, sku, name, quantity, unit_price_aud) VALUES ($1,$2,$3,$4,$5)`,
			orderID, it.SKU, it.Name, it.Qty, it.UnitPrice)
	}
	ordersCreated.Inc()
	jsonResponse(w, 201, map[string]any{"order_id": orderID, "total_aud": total, "status": "pending"})
}

func ordersHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/orders" || r.URL.Path == "/orders/" {
		if r.Method == http.MethodPost {
			createOrder(w, r)
		} else {
			listOrders(w, r)
		}
	} else if db != nil {
		getOrder(w, r)
	}
}

func seed(db *sql.DB) {
	db.Exec(`CREATE TABLE IF NOT EXISTS orders (
		id SERIAL PRIMARY KEY,
		customer_email TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending',
		total_aud NUMERIC(10,2),
		shipping_address TEXT,
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW()
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS order_items (
		id SERIAL PRIMARY KEY,
		order_id INT REFERENCES orders(id),
		sku TEXT NOT NULL,
		name TEXT NOT NULL,
		quantity INT NOT NULL DEFAULT 1,
		unit_price_aud NUMERIC(10,2) NOT NULL
	)`)

	var count int
	db.QueryRow(`SELECT COUNT(*) FROM orders`).Scan(&count)
	if count > 0 {
		return
	}

	seed := []struct {
		email   string
		status  string
		address string
		items   [][4]any
	}{
		{"sarah.chen@example.com.au", "delivered", "12 Harbour St, Sydney NSW 2000",
			[][4]any{{"RS-AS11-AU", "AirSense 11 AutoSet", 1, 1299.00}, {"RS-HH-AU", "HumidAir Heated Humidifier", 1, 199.00}}},
		{"james.wilson@example.com.au", "dispatched", "45 Collins St, Melbourne VIC 3000",
			[][4]any{{"RS-MINI-AU", "AirMini AutoSet", 1, 1099.00}, {"RS-TRAVEL-AU", "Travel Bag for AirMini", 1, 69.00}}},
		{"emily.nguyen@example.com.au", "pending", "8 Queen St, Brisbane QLD 4000",
			[][4]any{{"RS-F40-AU", "AirFit F40 Full Face Mask", 1, 189.00}, {"RS-FILTER-AU", "Ultra-Fine Filters Pack (6)", 2, 24.95}}},
		{"michael.brown@example.com.au", "delivered", "22 Rundle Mall, Adelaide SA 5000",
			[][4]any{{"RS-AS10-AU", "AirSense 10 AutoSet", 1, 999.00}, {"RS-N20-AU", "AirFit N20 Nasal Mask", 1, 159.00}, {"RS-CC-AU", "ClimateLineAir Heated Tube", 1, 129.00}}},
		{"lisa.park@example.com.au", "cancelled", "3 St Georges Tce, Perth WA 6000",
			[][4]any{{"RS-VAUTO-AU", "AirCurve 10 VAuto", 1, 1899.00}}},
		{"david.taylor@example.com.au", "processing", "100 King William St, Adelaide SA 5000",
			[][4]any{{"RS-P30-AU", "AirFit P30i Pillows Mask", 1, 169.00}, {"RS-MASK-CLEAN-AU", "SoClean 3 CPAP Cleaner", 1, 399.00}}},
	}

	for _, s := range seed {
		var total float64
		for _, it := range s.items {
			total += float64(it[2].(int)) * it[3].(float64)
		}
		var oid int
		db.QueryRow(`INSERT INTO orders (customer_email, status, total_aud, shipping_address) VALUES ($1,$2,$3,$4) RETURNING id`,
			s.email, s.status, total, s.address).Scan(&oid)
		for _, it := range s.items {
			db.Exec(`INSERT INTO order_items (order_id, sku, name, quantity, unit_price_aud) VALUES ($1,$2,$3,$4,$5)`,
				oid, it[0], it[1], it[2], it[3])
		}
	}
}

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://resmed:resmed@localhost:5432/orders?sslmode=disable"
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
	mux.HandleFunc("/orders", instrument(ordersHandler, "/orders"))
	mux.HandleFunc("/orders/", instrument(ordersHandler, "/orders/:id"))
	mux.Handle("/metrics", promhttp.Handler())

	log.Println("order-service listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
