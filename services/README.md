# Microservices — API Reference

Six Go microservices that power the ResMed platform. All services run on port `8080`, connect to PostgreSQL, and expose Prometheus metrics.

---

## Common endpoints (all services)

| Method | Path | Description |
|---|---|---|
| `GET` | `/health` | Returns `{"status":"healthy","service":"<name>"}` (200) or `{"status":"unhealthy","error":"..."}` (503) |
| `GET` | `/metrics` | Prometheus text format |

**Common Prometheus metrics:**
```
http_requests_total{method, path, status}
http_request_duration_seconds{method, path}
```

---

## device-catalog-api

Manages the ResMed product catalogue — CPAP machines, BiPAP machines, masks, and accessories.

**Default port:** 8080
**Database:** `device_catalog`
**Env var:** `DATABASE_URL` (default: `postgres://resmed:resmed@localhost:5432/device_catalog?sslmode=disable`)

### Endpoints

#### `GET /devices`
List all devices. Accepts optional `?category=` query param.

Categories: `cpap-machine`, `bipap-machine`, `mask`, `accessory`

```bash
curl http://localhost:8080/devices
curl http://localhost:8080/devices?category=mask
```

**Response 200:**
```json
[
  {
    "id": 1,
    "sku": "RS-AS11-AU",
    "name": "AirSense 11 AutoSet",
    "category": "cpap-machine",
    "price_aud": 1299.00,
    "description": "Auto-adjusting CPAP with integrated humidifier",
    "in_stock": true,
    "created_at": "2026-01-01T00:00:00Z"
  }
]
```

#### `GET /devices/:sku`
Get a single device by SKU.

```bash
curl http://localhost:8080/devices/RS-AS11-AU
```

**Response 404:**
```json
{"error": "device not found"}
```

---

## order-service

Customer order management with line items.

**Default port:** 8080
**Database:** `orders`
**Env var:** `DATABASE_URL` (default: `postgres://resmed:resmed@localhost:5432/orders?sslmode=disable`)

**Extra metric:** `orders_created_total` — incremented on each successful `POST /orders`

### Endpoints

#### `GET /orders`
List the 100 most recent orders. Accepts optional `?status=` filter.

Statuses: `pending`, `processing`, `dispatched`, `delivered`, `cancelled`

```bash
curl http://localhost:8080/orders
curl http://localhost:8080/orders?status=pending
```

**Response 200:**
```json
[
  {
    "id": 1,
    "customer_email": "sarah.chen@example.com.au",
    "status": "delivered",
    "total_aud": 1498.00,
    "shipping_address": "12 Harbour St, Sydney NSW 2000",
    "created_at": "2026-01-10T09:00:00Z",
    "updated_at": "2026-01-12T14:00:00Z"
  }
]
```

#### `POST /orders`
Create a new order.

```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{
    "customer_email": "new.patient@example.com",
    "shipping_address": "1 Test St, Sydney NSW 2000",
    "items": [
      {"sku": "RS-AS11-AU", "name": "AirSense 11 AutoSet", "quantity": 1, "unit_price_aud": 1299.00}
    ]
  }'
```

**Response 201:**
```json
{"order_id": 7, "total_aud": 1299.00, "status": "pending"}
```

#### `GET /orders/:id`
Get a single order with its line items.

```bash
curl http://localhost:8080/orders/1
```

**Response 200:** Order object with `items` array included.

---

## inventory-api

Warehouse stock levels for all ResMed SKUs.

**Default port:** 8080
**Database:** `inventory`
**Env var:** `DATABASE_URL` (default: `postgres://resmed:resmed@localhost:5432/inventory?sslmode=disable`)

### Endpoints

#### `GET /inventory`
List all inventory items with current stock levels.

```bash
curl http://localhost:8080/inventory
```

#### `GET /inventory/:sku`
Get stock level for a specific SKU.

```bash
curl http://localhost:8080/inventory/RS-AS11-AU
```

---

## patient-service

Patient records and CPAP device assignments.

**Default port:** 8080
**Database:** `patients`
**Env var:** `DATABASE_URL` (default: `postgres://resmed:resmed@localhost:5432/patients?sslmode=disable`)

### Endpoints

#### `GET /patients`
List all patients.

```bash
curl http://localhost:8080/patients
```

#### `GET /patients/:id`
Get a patient record including their device assignments.

```bash
curl http://localhost:8080/patients/1
```

---

## therapy-data-api

CPAP therapy sessions and compliance summaries per device.

**Default port:** 8080
**Database:** `therapy`
**Env var:** `DATABASE_URL` (default: `postgres://resmed:resmed@localhost:5432/therapy?sslmode=disable`)

### Endpoints

#### `GET /therapy`
List therapy sessions.

```bash
curl http://localhost:8080/therapy
```

#### `GET /therapy/compliance/:serial`
Get a compliance summary for a device by serial number.

```bash
curl http://localhost:8080/therapy/compliance/SN-12345
```

**Response 200:**
```json
{
  "serial": "SN-12345",
  "total_sessions": 45,
  "compliant_sessions": 40,
  "compliance_pct": 88.9,
  "avg_usage_hours": 7.2
}
```

---

## notification-service

Async notification dispatch for patient and order events.

**Default port:** 8080
**Database:** `notifications`
**Env var:** `DATABASE_URL` (default: `postgres://resmed:resmed@localhost:5432/notifications?sslmode=disable`)

**Background worker:** polls for `status='pending'` notifications every 30 seconds, processes up to 10 per cycle.

### Endpoints

#### `GET /notifications`
List notifications.

```bash
curl http://localhost:8080/notifications
```

---

## Running locally

Each service requires a Postgres instance. The quickest way to get one:

```bash
docker run -d \
  --name resmed-pg \
  -e POSTGRES_USER=resmed \
  -e POSTGRES_PASSWORD=resmed \
  -e POSTGRES_DB=device_catalog \
  -p 5432:5432 \
  postgres:16-alpine
```

Then:
```bash
cd services/device-catalog-api
go run .
```

All services seed their own tables on first startup via `seed()` in `main.go`.

## Running tests

```bash
cd services/<name>
go test ./... -v -coverprofile=coverage.out
go tool cover -func=coverage.out
```

Coverage must be ≥ 70% — enforced in CI.
