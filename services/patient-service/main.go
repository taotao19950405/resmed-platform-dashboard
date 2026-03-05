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

	activePatients = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "patients_active_total",
		Help: "Number of patients with active device assignments",
	})
)

func init() {
	prometheus.MustRegister(httpRequests, httpDuration, activePatients)
}

type Patient struct {
	ID        int    `json:"id"`
	MRN       string `json:"mrn"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	DOB       string `json:"dob"`
	State     string `json:"state"`
	CreatedAt string `json:"created_at"`
}

type DeviceAssignment struct {
	ID           int    `json:"id"`
	PatientID    int    `json:"patient_id"`
	SerialNumber string `json:"serial_number"`
	SKU          string `json:"sku"`
	ModelName    string `json:"model_name"`
	TherapyStart string `json:"therapy_start"`
	Active       bool   `json:"active"`
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
	jsonResponse(w, 200, map[string]string{"status": "healthy", "service": "patient-service"})
}

func listPatients(w http.ResponseWriter, r *http.Request) {
	if db == nil { jsonResponse(w, 500, map[string]string{"error": "db not initialized"}); return }
	rows, err := db.Query(`SELECT id, mrn, first_name, last_name, dob, state, created_at FROM patients ORDER BY last_name LIMIT 100`)
	if err != nil {
		jsonResponse(w, 500, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()
	patients := []Patient{}
	for rows.Next() {
		var p Patient
		rows.Scan(&p.ID, &p.MRN, &p.FirstName, &p.LastName, &p.DOB, &p.State, &p.CreatedAt)
		patients = append(patients, p)
	}
	jsonResponse(w, 200, patients)
}

func getPatient(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/patients/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		jsonResponse(w, 400, map[string]string{"error": "invalid patient id"})
		return
	}
	var p Patient
	if db == nil { jsonResponse(w, 500, map[string]string{"error": "db not initialized"}); return }
	err = db.QueryRow(`SELECT id, mrn, first_name, last_name, dob, state, created_at FROM patients WHERE id=$1`, id).
		Scan(&p.ID, &p.MRN, &p.FirstName, &p.LastName, &p.DOB, &p.State, &p.CreatedAt)
	if err == sql.ErrNoRows {
		jsonResponse(w, 404, map[string]string{"error": "patient not found"})
		return
	}
	if err != nil {
		jsonResponse(w, 500, map[string]string{"error": err.Error()})
		return
	}
	rows, _ := db.Query(`SELECT id, patient_id, serial_number, sku, model_name, therapy_start, active FROM device_assignments WHERE patient_id=$1`, p.ID)
	defer rows.Close()
	assignments := []DeviceAssignment{}
	for rows.Next() {
		var a DeviceAssignment
		rows.Scan(&a.ID, &a.PatientID, &a.SerialNumber, &a.SKU, &a.ModelName, &a.TherapyStart, &a.Active)
		assignments = append(assignments, a)
	}
	jsonResponse(w, 200, map[string]any{"patient": p, "device_assignments": assignments})
}

func patientsHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/patients" || r.URL.Path == "/patients/" {
		listPatients(w, r)
	} else {
		getPatient(w, r)
	}
}

func seed(db *sql.DB) {
	db.Exec(`CREATE TABLE IF NOT EXISTS patients (
		id SERIAL PRIMARY KEY,
		mrn TEXT UNIQUE NOT NULL,
		first_name TEXT NOT NULL,
		last_name TEXT NOT NULL,
		dob DATE NOT NULL,
		state TEXT NOT NULL,
		created_at TIMESTAMPTZ DEFAULT NOW()
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS device_assignments (
		id SERIAL PRIMARY KEY,
		patient_id INT REFERENCES patients(id),
		serial_number TEXT UNIQUE NOT NULL,
		sku TEXT NOT NULL,
		model_name TEXT NOT NULL,
		therapy_start DATE NOT NULL,
		active BOOLEAN DEFAULT true
	)`)

	var count int
	db.QueryRow(`SELECT COUNT(*) FROM patients`).Scan(&count)
	if count > 0 {
		updateActivePatients()
		return
	}

	patients := [][]any{
		{"RMD-001423", "Sarah", "Chen", "1978-04-12", "NSW"},
		{"RMD-001424", "James", "Wilson", "1965-09-23", "VIC"},
		{"RMD-001425", "Emily", "Nguyen", "1982-11-05", "QLD"},
		{"RMD-001426", "Michael", "Brown", "1971-03-18", "SA"},
		{"RMD-001427", "Lisa", "Park", "1989-07-30", "WA"},
		{"RMD-001428", "David", "Taylor", "1955-12-01", "NSW"},
		{"RMD-001429", "Anna", "Martinez", "1993-06-14", "VIC"},
		{"RMD-001430", "Robert", "Johnson", "1948-08-22", "QLD"},
	}
	assignments := [][]any{
		{1, "AS11-AU-000142", "RS-AS11-AU", "AirSense 11 AutoSet", "2024-01-15"},
		{2, "MINI-AU-000287", "RS-MINI-AU", "AirMini AutoSet", "2023-08-20"},
		{3, "AS10-AU-000391", "RS-AS10-AU", "AirSense 10 AutoSet", "2023-11-03"},
		{4, "VAUTO-AU-000104", "RS-VAUTO-AU", "AirCurve 10 VAuto", "2022-05-17"},
		{5, "AS11-AU-000523", "RS-AS11-AU", "AirSense 11 AutoSet", "2024-03-01"},
		{6, "AS10-AU-000612", "RS-AS10-AU", "AirSense 10 AutoSet", "2021-09-10"},
		{7, "MINI-AU-000734", "RS-MINI-AU", "AirMini AutoSet", "2024-06-22"},
		{8, "VAUTO-AU-000845", "RS-VAUTO-AU", "AirCurve 10 VAuto", "2020-12-05"},
	}
	for _, p := range patients {
		db.Exec(`INSERT INTO patients (mrn, first_name, last_name, dob, state) VALUES ($1,$2,$3,$4,$5) ON CONFLICT (mrn) DO NOTHING`,
			p[0], p[1], p[2], p[3], p[4])
	}
	for _, a := range assignments {
		db.Exec(`INSERT INTO device_assignments (patient_id, serial_number, sku, model_name, therapy_start) VALUES ($1,$2,$3,$4,$5) ON CONFLICT (serial_number) DO NOTHING`,
			a[0], a[1], a[2], a[3], a[4])
	}
	updateActivePatients()
}

func updateActivePatients() {
	var count int
	db.QueryRow(`SELECT COUNT(DISTINCT patient_id) FROM device_assignments WHERE active=true`).Scan(&count)
	activePatients.Set(float64(count))
}

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://resmed:resmed@localhost:5432/patients?sslmode=disable"
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
	mux.HandleFunc("/patients", instrument(patientsHandler, "/patients"))
	mux.HandleFunc("/patients/", instrument(patientsHandler, "/patients/:id"))
	mux.Handle("/metrics", promhttp.Handler())

	log.Println("patient-service listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
