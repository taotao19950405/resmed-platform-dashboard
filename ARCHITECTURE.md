# Architecture & Technical Specification

**Project:** ResMed Platform Dashboard
**Model:** claude-sonnet-4-6 (`claude-sonnet-4-6`)
**Last updated:** 2026-03-05

---

## 1. Overview

The ResMed Platform Dashboard is an internal developer portal built on [Backstage](https://backstage.io). It provides engineering teams with a unified view of:

- **Service health** — live HTTP health check status for all backend microservices
- **Kubernetes deployments** — pod counts, replica status, and readiness per service
- **Operational metrics** — Prometheus-scraped request rates, latencies, and business counters

The system consists of three layers:

1. **Backstage frontend** — custom `platform-dashboard` plugin (React + Material UI)
2. **Go microservices** — six domain services, each with PostgreSQL persistence and Prometheus metrics
3. **Kubernetes** — all services run in a `production` namespace; a shared PostgreSQL instance hosts six separate databases

---

## 2. Technology stack

| Layer | Technology | Version |
|---|---|---|
| Developer portal | Backstage | latest |
| Plugin language | TypeScript / React | 18 |
| Microservice language | Go | 1.26 |
| Database | PostgreSQL | 16 |
| Container runtime | Docker | buildx |
| Orchestration | Kubernetes | 1.29+ |
| Metrics | Prometheus + ServiceMonitor CRD | operator v0.70+ |
| CI | GitHub Actions | — |
| Image registry | GitHub Container Registry (GHCR) | — |
| Lint | golangci-lint | v1.64 |
| Vulnerability scan | Trivy | latest |
| K8s manifest validation | kubeconform | latest |
| Dependency automation | Dependabot | weekly |

---

## 3. Microservice specifications

All six services follow the same structure:

```
services/<name>/
├── main.go          # HTTP server, DB init, seed data, Prometheus metrics
├── main_test.go     # Unit + integration tests (≥70% coverage)
├── Dockerfile       # Multi-stage: golang:1.26-alpine builder → alpine:3.20 runtime
├── go.mod
└── go.sum
```

### 3.1 device-catalog-api

Manages the ResMed product catalogue (CPAP machines, BiPAP machines, masks, accessories).

**Database:** `device_catalog`

**Schema:**
```sql
CREATE TABLE devices (
  id          SERIAL PRIMARY KEY,
  sku         TEXT UNIQUE NOT NULL,
  name        TEXT NOT NULL,
  category    TEXT NOT NULL,          -- cpap-machine | bipap-machine | mask | accessory
  price_aud   NUMERIC(10,2) NOT NULL,
  description TEXT,
  in_stock    BOOLEAN DEFAULT true,
  created_at  TIMESTAMPTZ DEFAULT NOW()
);
```

**Endpoints:**

| Method | Path | Description |
|---|---|---|
| GET | `/health` | Liveness/readiness probe |
| GET | `/devices` | List all devices; optional `?category=` filter |
| GET | `/devices/:sku` | Get single device by SKU |
| GET | `/metrics` | Prometheus metrics |

**Seed data:** 14 real ResMed SKUs (AirSense 11, AirMini, AirCurve 10 VAuto, AirFit masks, accessories).

---

### 3.2 order-service

Manages customer orders for ResMed equipment, including line items.

**Database:** `orders`

**Schema:**
```sql
CREATE TABLE orders (
  id               SERIAL PRIMARY KEY,
  customer_email   TEXT NOT NULL,
  status           TEXT NOT NULL DEFAULT 'pending',  -- pending | processing | dispatched | delivered | cancelled
  total_aud        NUMERIC(10,2),
  shipping_address TEXT,
  created_at       TIMESTAMPTZ DEFAULT NOW(),
  updated_at       TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE order_items (
  id             SERIAL PRIMARY KEY,
  order_id       INT REFERENCES orders(id),
  sku            TEXT NOT NULL,
  name           TEXT NOT NULL,
  quantity       INT NOT NULL DEFAULT 1,
  unit_price_aud NUMERIC(10,2) NOT NULL
);
```

**Endpoints:**

| Method | Path | Description |
|---|---|---|
| GET | `/health` | Liveness/readiness probe |
| GET | `/orders` | List orders (last 100); optional `?status=` filter |
| POST | `/orders` | Create order with line items |
| GET | `/orders/:id` | Get order with items by ID |
| GET | `/metrics` | Prometheus metrics incl. `orders_created_total` |

**Extra metric:** `orders_created_total` counter incremented on each successful `POST /orders`.

---

### 3.3 inventory-api

Tracks warehouse stock levels for all SKUs.

**Database:** `inventory`

**Endpoints:**

| Method | Path | Description |
|---|---|---|
| GET | `/health` | Liveness/readiness probe |
| GET | `/inventory` | List all inventory items |
| GET | `/inventory/:sku` | Get stock level for a specific SKU |
| GET | `/metrics` | Prometheus metrics |

---

### 3.4 patient-service

Manages patient records and their CPAP device assignments.

**Database:** `patients`

**Endpoints:**

| Method | Path | Description |
|---|---|---|
| GET | `/health` | Liveness/readiness probe |
| GET | `/patients` | List all patients |
| GET | `/patients/:id` | Get patient with device assignments |
| GET | `/metrics` | Prometheus metrics |

---

### 3.5 therapy-data-api

Stores CPAP therapy session data and calculates compliance summaries.

**Database:** `therapy`

**Endpoints:**

| Method | Path | Description |
|---|---|---|
| GET | `/health` | Liveness/readiness probe |
| GET | `/therapy` | List therapy sessions |
| GET | `/therapy/compliance/:serial` | Compliance summary for a device serial number |
| GET | `/metrics` | Prometheus metrics |

---

### 3.6 notification-service

Handles async dispatch of patient and order notifications (email/SMS triggers).

**Database:** `notifications`

**Endpoints:**

| Method | Path | Description |
|---|---|---|
| GET | `/health` | Liveness/readiness probe |
| GET | `/notifications` | List notifications |
| GET | `/metrics` | Prometheus metrics |

**Background worker:** polls `notifications` table every 30s for `status='pending'` records and processes up to 10 at a time.

---

## 4. Common service patterns

Every service implements:

### Health check
```json
GET /health
→ 200 {"status": "healthy", "service": "<name>"}
→ 503 {"status": "unhealthy", "error": "<reason>"}
```
Health is gated on a successful `db.Ping()`. Kubernetes readiness and liveness probes both target `/health`.

### Prometheus instrumentation
```
http_requests_total{method, path, status}    counter
http_request_duration_seconds{method, path}  histogram (default buckets)
```
Domain-specific counters added per service (e.g. `orders_created_total`).

### PostgreSQL connection retry
Services retry the DB connection up to 10 times with a 3-second delay to tolerate pod startup ordering.

### Dockerfile (multi-stage)
```dockerfile
FROM golang:1.26-alpine AS builder
# → CGO_ENABLED=0 GOOS=linux go build -o service .

FROM alpine:3.20
# → HEALTHCHECK via wget /health
# → EXPOSE 8080
```

---

## 5. Kubernetes topology

```
Namespace: production
│
├── postgres (Deployment, 1 replica)
│   ├── image: postgres:16-alpine
│   ├── init ConfigMap: creates 6 databases + resmed user
│   ├── Secret: POSTGRES_PASSWORD
│   └── Service: postgres:5432
│
├── device-catalog-api  (Deployment, 2 replicas)
├── order-service       (Deployment, 2 replicas)
├── inventory-api       (Deployment, 2 replicas)
├── patient-service     (Deployment, 2 replicas)
├── therapy-data-api    (Deployment, 2 replicas)
└── notification-service (Deployment, 2 replicas)
    │
    └── each has:
        ├── readinessProbe:  GET /health  (delay 10s, period 10s)
        ├── livenessProbe:   GET /health  (delay 20s, period 30s)
        ├── resources:  requests 64Mi/50m  limits 128Mi/200m
        ├── annotation: prometheus.io/scrape="true" port="8080" path="/metrics"
        └── env DATABASE_URL → postgres.production.svc.cluster.local:5432/<db>

Namespace: monitoring
└── ServiceMonitor: scrapes production/* on /metrics every 15s
```

All services connect to the shared PostgreSQL instance using separate databases. The connection string pattern is:
```
postgres://resmed:resmed@postgres.production.svc.cluster.local:5432/<dbname>?sslmode=disable
```

---

## 6. Backstage plugin — platform-dashboard

### Purpose
Provides a single-page dashboard inside Backstage showing:
- **Service Health Table** — sortable table with health status, latency indicator, and links for all backend services
- **K8s Deployment Panel** — expandable cards per deployment showing pod names, readiness, and restart counts

### Plugin structure
```
plugins/platform-dashboard/src/
├── types/index.ts                    # HealthStatus, ServiceHealth, Pod, K8sDeployment
├── api/
│   ├── PlatformDashboardApi.ts       # TypeScript interface + ApiRef
│   ├── MockPlatformDashboardClient.ts # Mock (8 services, 4 deployments, 300–600ms delay)
│   └── PlatformDashboardClient.ts    # Production stub (to be wired to real backend)
├── hooks/
│   ├── useServiceHealth.ts           # Polls getServiceHealth(), returns loading/error/data
│   └── useK8sDeployments.ts          # Polls getK8sDeployments(), returns loading/error/data
├── components/
│   ├── StatusChip/                   # Coloured chip: healthy=green, degraded=amber, down=red
│   ├── ServiceHealthTable/           # Sortable MUI table with StatusChip + icon links
│   ├── K8sDeploymentPanel/           # Expand/collapse pods per deployment
│   └── PlatformDashboardPage/        # Root page component
├── plugin.ts                         # createPlugin + createApiFactory (MockClient)
└── index.ts                          # Public exports
```

### API interface
```typescript
interface PlatformDashboardApi {
  getServiceHealth(): Promise<ServiceHealth[]>;
  getK8sDeployments(): Promise<K8sDeployment[]>;
}
```

### Route
Registered at `/platform-dashboard` in `packages/app/src/App.tsx`, with a sidebar link in `packages/app/src/components/Root/Root.tsx`.

---

## 7. CI/CD pipeline

### CI (`.github/workflows/ci.yml`)

Triggered on every push and pull request to `main`.

```
┌─────────┐  ┌──────┐  ┌───────┐  ┌──────┐  ┌─────────────┐
│  test   │  │ lint │  │ build │→ │ scan │  │ validate-k8s│
│ (×6 svc)│  │(×6)  │  │ (×6)  │  │ (×6) │  │             │
└────┬────┘  └──┬───┘  └───────┘  └──┬───┘  └──────┬──────┘
     └──────────┴────────────────────┴──────────────┘
                              ▼
                       ci-passed (gate)
```

| Job | Tool | Quality gate |
|---|---|---|
| `test` | `go test ./... -coverprofile` | ≥70% total coverage per service |
| `lint` | golangci-lint v1.64 + `.golangci.yml` | zero lint errors |
| `build` | docker/build-push-action (push:false) | image builds successfully |
| `scan` | Trivy | no unfixed CRITICAL CVEs |
| `validate-k8s` | kubeconform --strict | all manifests valid (ServiceMonitors excluded) |
| `ci-passed` | bash gate check | all 5 jobs must succeed |

All jobs use a matrix strategy across all 6 services with `fail-fast: false` (one failure does not cancel siblings).

### CD (`.github/workflows/cd.yml`)

Triggered on push to `main` when `services/**` or `k8s/**` changes.

```
push-images (×6, parallel)
    → GHCR: ghcr.io/<owner>/<service>:sha-<short>  (+ :latest on main)
         ↓
update-manifests
    → sed image tags in k8s/services/*.yaml
    → git commit + push "chore: update image tags to <sha>"
         ↓
deploy (self-hosted runner, environment: production)
    → kubectl apply namespaces → postgres → services → monitoring
    → kubectl rollout status (180s timeout per service)
    → smoke test: port-forward device-catalog-api → GET /health
```

### Dependabot
Weekly automated PRs for:
- Go modules in each of the 6 services
- GitHub Actions action versions

---

## 8. Monitoring

Prometheus scrapes every pod in the `production` namespace via the `ServiceMonitor` CRD (interval: 15s, path: `/metrics`).

Key metrics to alert on:

| Metric | Type | Alert condition |
|---|---|---|
| `http_requests_total` | Counter | Error rate (`status=~"5.."`) > 1% over 5m |
| `http_request_duration_seconds` | Histogram | p99 > 2s over 5m |
| `orders_created_total` | Counter | Rate drops to 0 for > 10m |
| Kubernetes `kube_deployment_status_replicas_unavailable` | Gauge | > 0 for > 2m |

---

## 9. Local development guide

### Starting all services with Docker Compose (optional)

Each service can be run standalone with:
```bash
cd services/<name>
DATABASE_URL=postgres://resmed:resmed@localhost:5432/<db>?sslmode=disable go run .
```

### Running tests
```bash
cd services/<name>
go test ./... -v -coverprofile=coverage.out
go tool cover -html=coverage.out   # view in browser
```

### Running lint
```bash
cd services/<name>
golangci-lint run --config ../../.golangci.yml
```

---

## 10. AI tooling

This project was developed with [Claude Code](https://claude.ai/claude-code) using **Claude Sonnet 4.6** (`claude-sonnet-4-6`). All code generation, architecture decisions, and documentation were produced in a series of Claude Code sessions.

For AI-assisted development, the recommended model is:
- **claude-sonnet-4-6** — best balance of capability and speed for code generation tasks
- **claude-opus-4-6** — for complex architectural reasoning and deep refactoring
