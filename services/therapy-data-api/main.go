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

	avgAHI = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "therapy_avg_ahi_score",
		Help: "Average AHI score across all active patients (last 30 days)",
	})

	complianceRate = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "therapy_compliance_rate",
		Help: "Percentage of patients using device >= 4 hours/night (CMS compliance)",
	})
)

func init() {
	prometheus.MustRegister(httpRequests, httpDuration, avgAHI, complianceRate)
}

type TherapySession struct {
	ID           int     `json:"id"`
	SerialNumber string  `json:"serial_number"`
	SessionDate  string  `json:"session_date"`
	UsageHours   float64 `json:"usage_hours"`
	AHI          float64 `json:"ahi"`
	LeakRate     float64 `json:"leak_rate_l_min"`
	Pressure     float64 `json:"pressure_cmh2o"`
}

type ComplianceSummary struct {
	SerialNumber    string  `json:"serial_number"`
	AvgUsageHours   float64 `json:"avg_usage_hours"`
	AvgAHI          float64 `json:"avg_ahi"`
	CompliantNights int     `json:"compliant_nights"`
	TotalNights     int     `json:"total_nights"`
	ComplianceRate  float64 `json:"compliance_rate_pct"`
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
	jsonResponse(w, 200, map[string]string{"status": "healthy", "service": "therapy-data-api"})
}

func listSessions(w http.ResponseWriter, r *http.Request) {
	serial := r.URL.Query().Get("serial")
	days := r.URL.Query().Get("days")
	if days == "" {
		days = "30"
	}
	d, _ := strconv.Atoi(days)
	var rows *sql.Rows
	var err error
	if serial != "" {
		rows, err = db.Query(`SELECT id, serial_number, session_date, usage_hours, ahi, leak_rate, pressure FROM therapy_sessions WHERE serial_number=$1 AND session_date >= NOW() - ($2 || ' days')::INTERVAL ORDER BY session_date DESC`, serial, d)
	} else if db != nil {
		rows, err = db.Query(`SELECT id, serial_number, session_date, usage_hours, ahi, leak_rate, pressure FROM therapy_sessions WHERE session_date >= NOW() - ($1 || ' days')::INTERVAL ORDER BY session_date DESC LIMIT 200`, d)
	}
	if err != nil {
		jsonResponse(w, 500, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()
	sessions := []TherapySession{}
	for rows.Next() {
		var s TherapySession
		rows.Scan(&s.ID, &s.SerialNumber, &s.SessionDate, &s.UsageHours, &s.AHI, &s.LeakRate, &s.Pressure)
		sessions = append(sessions, s)
	}
	jsonResponse(w, 200, sessions)
}

func getCompliance(w http.ResponseWriter, r *http.Request) {
	serial := r.URL.Path[len("/therapy/compliance/"):]
	var c ComplianceSummary
	c.SerialNumber = serial
	if db == nil { jsonResponse(w, 500, map[string]string{"error": "db not initialized"}); return }
	err := db.QueryRow(`
		SELECT
			ROUND(AVG(usage_hours)::numeric, 2),
			ROUND(AVG(ahi)::numeric, 2),
			COUNT(*) FILTER (WHERE usage_hours >= 4),
			COUNT(*)
		FROM therapy_sessions
		WHERE serial_number=$1 AND session_date >= NOW() - INTERVAL '30 days'`, serial).
		Scan(&c.AvgUsageHours, &c.AvgAHI, &c.CompliantNights, &c.TotalNights)
	if err != nil {
		jsonResponse(w, 500, map[string]string{"error": err.Error()})
		return
	}
	if c.TotalNights > 0 {
		c.ComplianceRate = float64(c.CompliantNights) / float64(c.TotalNights) * 100
	}
	jsonResponse(w, 200, c)
}

func therapyHandler(w http.ResponseWriter, r *http.Request) {
	listSessions(w, r)
}

func updateMetrics() {
	db.QueryRow(`SELECT COALESCE(AVG(ahi),0) FROM therapy_sessions WHERE session_date >= NOW() - INTERVAL '30 days'`).Scan(new(float64))
	var ahi float64
	db.QueryRow(`SELECT COALESCE(ROUND(AVG(ahi)::numeric,2),0) FROM therapy_sessions WHERE session_date >= NOW() - INTERVAL '30 days'`).Scan(&ahi)
	avgAHI.Set(ahi)

	var compliant, total int
	db.QueryRow(`SELECT COUNT(*) FILTER (WHERE avg_hours >= 4), COUNT(*) FROM (SELECT serial_number, AVG(usage_hours) as avg_hours FROM therapy_sessions WHERE session_date >= NOW() - INTERVAL '30 days' GROUP BY serial_number) t`).Scan(&compliant, &total)
	if total > 0 {
		complianceRate.Set(float64(compliant) / float64(total) * 100)
	}
}

func seed(db *sql.DB) {
	db.Exec(`CREATE TABLE IF NOT EXISTS therapy_sessions (
		id SERIAL PRIMARY KEY,
		serial_number TEXT NOT NULL,
		session_date DATE NOT NULL,
		usage_hours NUMERIC(4,2) NOT NULL,
		ahi NUMERIC(5,2) NOT NULL,
		leak_rate NUMERIC(5,2),
		pressure NUMERIC(4,1),
		UNIQUE(serial_number, session_date)
	)`)

	var count int
	db.QueryRow(`SELECT COUNT(*) FROM therapy_sessions`).Scan(&count)
	if count > 0 {
		updateMetrics()
		return
	}

	serials := []string{
		"AS11-AU-000142", "MINI-AU-000287", "AS10-AU-000391",
		"VAUTO-AU-000104", "AS11-AU-000523", "AS10-AU-000612",
		"MINI-AU-000734", "VAUTO-AU-000845",
	}
	ahiBase := []float64{2.1, 8.4, 1.8, 3.2, 5.6, 12.1, 2.9, 4.7}
	usageBase := []float64{7.2, 3.1, 6.8, 5.5, 4.2, 2.8, 7.1, 6.3}

	for i, serial := range serials {
		for day := 0; day < 30; day++ {
			date := time.Now().AddDate(0, 0, -day).Format("2006-01-02")
			ahi := ahiBase[i] + (float64(day%3)-1)*0.3
			usage := usageBase[i] + (float64(day%5)-2)*0.2
			if usage < 0 {
				usage = 0.5
			}
			leak := 8.0 + float64(i)*1.2
			pressure := 8.5 + float64(i)*0.5
			db.Exec(`INSERT INTO therapy_sessions (serial_number, session_date, usage_hours, ahi, leak_rate, pressure) VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT DO NOTHING`,
				serial, date, usage, ahi, leak, pressure)
		}
	}
	updateMetrics()
}

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://resmed:resmed@localhost:5432/therapy?sslmode=disable"
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
		for range time.Tick(60 * time.Second) {
			updateMetrics()
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", instrument(healthHandler, "/health"))
	mux.HandleFunc("/therapy", instrument(therapyHandler, "/therapy"))
	mux.HandleFunc("/therapy/compliance/", instrument(getCompliance, "/therapy/compliance/:serial"))
	mux.Handle("/metrics", promhttp.Handler())

	log.Println("therapy-data-api listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
