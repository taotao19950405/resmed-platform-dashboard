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

	notificationsSent = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "notifications_sent_total",
		Help: "Total notifications sent by type",
	}, []string{"type"})

	pendingNotifications = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "notifications_pending_total",
		Help: "Number of notifications in pending state",
	})
)

func init() {
	prometheus.MustRegister(httpRequests, httpDuration, notificationsSent, pendingNotifications)
}

type Notification struct {
	ID        int    `json:"id"`
	Type      string `json:"type"`
	Recipient string `json:"recipient"`
	Subject   string `json:"subject"`
	Payload   string `json:"payload"`
	Status    string `json:"status"`
	SentAt    string `json:"sent_at,omitempty"`
	CreatedAt string `json:"created_at"`
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

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		jsonResponse(w, 503, map[string]string{"status": "unhealthy", "error": "db not initialized"})
		return
	}
	if err := db.Ping(); err != nil {
		jsonResponse(w, 503, map[string]string{"status": "unhealthy", "error": err.Error()})
		return
	}
	jsonResponse(w, 200, map[string]string{"status": "healthy", "service": "notification-service"})
}

func listNotifications(w http.ResponseWriter, r *http.Request) {
	nType := r.URL.Query().Get("type")
	status := r.URL.Query().Get("status")
	query := `SELECT id, type, recipient, subject, payload, status, COALESCE(sent_at::text,''), created_at FROM notifications WHERE 1=1`
	args := []any{}
	i := 1
	if nType != "" {
		query += ` AND type=$` + strconv.Itoa(i)
		args = append(args, nType)
		i++
	}
	if status != "" {
		query += ` AND status=$` + strconv.Itoa(i)
		args = append(args, status)
	}
	query += ` ORDER BY created_at DESC LIMIT 100`
	if db == nil { jsonResponse(w, 500, map[string]string{"error": "db not initialized"}); return }
	rows, err := db.Query(query, args...)
	if err != nil {
		jsonResponse(w, 500, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()
	notifications := []Notification{}
	for rows.Next() {
		var n Notification
		rows.Scan(&n.ID, &n.Type, &n.Recipient, &n.Subject, &n.Payload, &n.Status, &n.SentAt, &n.CreatedAt)
		notifications = append(notifications, n)
	}
	jsonResponse(w, 200, notifications)
}

func createNotification(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Type      string `json:"type"`
		Recipient string `json:"recipient"`
		Subject   string `json:"subject"`
		Payload   string `json:"payload"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, 400, map[string]string{"error": "invalid body"})
		return
	}
	var id int
	if db == nil { jsonResponse(w, 500, map[string]string{"error": "db not initialized"}); return }
	err := db.QueryRow(`INSERT INTO notifications (type, recipient, subject, payload, status) VALUES ($1,$2,$3,$4,'pending') RETURNING id`,
		req.Type, req.Recipient, req.Subject, req.Payload).Scan(&id)
	if err != nil {
		jsonResponse(w, 500, map[string]string{"error": err.Error()})
		return
	}
	updatePendingGauge()
	jsonResponse(w, 201, map[string]any{"id": id, "status": "pending"})
}

func notificationsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		createNotification(w, r)
	} else if db != nil {
		listNotifications(w, r)
	}
}

func updatePendingGauge() {
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE status='pending'`).Scan(&count)
	pendingNotifications.Set(float64(count))
}

func processNotifications() {
	if db == nil { return }
	rows, err := db.Query(`SELECT id, type FROM notifications WHERE status='pending' LIMIT 10`)
	if err != nil {
		return
	}
	defer rows.Close()
	var ids []int
	var types []string
	for rows.Next() {
		var id int
		var t string
		rows.Scan(&id, &t)
		ids = append(ids, id)
		types = append(types, t)
	}
	for i, id := range ids {
		db.Exec(`UPDATE notifications SET status='sent', sent_at=NOW() WHERE id=$1`, id)
		notificationsSent.WithLabelValues(types[i]).Inc()
	}
	updatePendingGauge()
}

func seed(db *sql.DB) {
	db.Exec(`CREATE TABLE IF NOT EXISTS notifications (
		id SERIAL PRIMARY KEY,
		type TEXT NOT NULL,
		recipient TEXT NOT NULL,
		subject TEXT NOT NULL,
		payload TEXT,
		status TEXT NOT NULL DEFAULT 'pending',
		sent_at TIMESTAMPTZ,
		created_at TIMESTAMPTZ DEFAULT NOW()
	)`)

	var count int
	db.QueryRow(`SELECT COUNT(*) FROM notifications`).Scan(&count)
	if count > 0 {
		updatePendingGauge()
		return
	}

	notifications := [][]any{
		{"low_stock", "warehouse@resmed.com.au", "Low Stock Alert: AirFit F20 Full Face Mask", `{"sku":"RS-F20-AU","quantity":7,"threshold":30}`},
		{"low_stock", "warehouse@resmed.com.au", "Low Stock Alert: AirFit P30i Pillows Mask", `{"sku":"RS-P30-AU","quantity":4,"threshold":30}`},
		{"therapy_non_compliance", "clinician@resmed.com.au", "Patient RMD-001424 Therapy Non-Compliance", `{"mrn":"RMD-001424","avg_hours":3.1,"threshold":4.0}`},
		{"therapy_non_compliance", "clinician@resmed.com.au", "Patient RMD-001428 Therapy Non-Compliance", `{"mrn":"RMD-001428","avg_hours":2.8,"threshold":4.0}`},
		{"order_dispatched", "james.wilson@example.com.au", "Your ResMed Order #2 Has Shipped", `{"order_id":2,"tracking":"AUS123456789"}`},
		{"order_delivered", "sarah.chen@example.com.au", "Your ResMed Order #1 Has Been Delivered", `{"order_id":1}`},
		{"device_warranty", "michael.brown@example.com.au", "AirCurve 10 VAuto Warranty Expiring Soon", `{"serial":"VAUTO-AU-000104","expiry":"2025-05-17"}`},
		{"mask_replacement", "emily.nguyen@example.com.au", "Time to Replace Your CPAP Mask Cushion", `{"serial":"AS10-AU-000391","last_replaced":"2024-09-03"}`},
	}

	statuses := []string{"sent", "sent", "sent", "pending", "sent", "sent", "pending", "sent"}
	for i, n := range notifications {
		if statuses[i] == "sent" {
			db.Exec(`INSERT INTO notifications (type, recipient, subject, payload, status, sent_at) VALUES ($1,$2,$3,$4,'sent', NOW() - ($5 || ' hours')::INTERVAL)`,
				n[0], n[1], n[2], n[3], (i+1)*3)
		} else {
			db.Exec(`INSERT INTO notifications (type, recipient, subject, payload, status) VALUES ($1,$2,$3,$4,'pending')`,
				n[0], n[1], n[2], n[3])
		}
	}
	updatePendingGauge()
}

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://resmed:resmed@localhost:5432/notifications?sslmode=disable"
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
			processNotifications()
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", instrument(healthHandler, "/health"))
	mux.HandleFunc("/notifications", instrument(notificationsHandler, "/notifications"))
	mux.Handle("/metrics", promhttp.Handler())

	log.Println("notification-service listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
