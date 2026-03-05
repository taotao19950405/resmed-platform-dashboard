# ResMed Platform Dashboard

A [Backstage](https://backstage.io) internal developer portal with a custom **Platform Dashboard** plugin, backed by six Go microservices, PostgreSQL, Kubernetes, and a full GitHub Actions CI/CD pipeline.

---

## Project structure

```
resmed-platform-dashboard/
│
│  ← The developer portal (what you see in the browser at localhost:3000)
├── packages/
│   ├── app/                        # React frontend — the Backstage UI
│   └── backend/                    # Node.js backend — serves UI + APIs
│
│  ← Your custom plugin (lives inside the portal)
├── plugins/
│   └── platform-dashboard/         # Shows service health + K8s pod status
│
│  ← The 6 Go microservices (the actual product backend)
├── services/
│   ├── device-catalog-api/         # ResMed device catalogue (CRUD + stock)
│   ├── order-service/              # Customer order management
│   ├── inventory-api/              # Warehouse stock levels
│   ├── patient-service/            # Patient records & device assignments
│   ├── therapy-data-api/           # CPAP therapy sessions & compliance
│   └── notification-service/       # Async notification dispatch
│
│  ← Kubernetes config (how services run in production)
├── k8s/
│   ├── namespaces.yaml             # Creates "production" namespace
│   ├── postgres/                   # One shared PostgreSQL pod, 6 databases inside
│   ├── services/                   # Deployment + Service per microservice
│   └── monitoring/                 # Prometheus ServiceMonitor CRD
│
│  ← CI/CD (automated quality checks + deployment)
└── .github/
    ├── workflows/
    │   ├── ci.yml                  # Runs on every push/PR — quality gates
    │   └── cd.yml                  # Runs on merge to main — build & deploy
    └── dependabot.yml              # Auto-updates dependencies weekly
```

---

## How the two backends are different

There are **two completely separate backends** in this repo — a common point of confusion:

| | Backstage backend (`packages/backend`) | Microservices (`services/`) |
|---|---|---|
| Language | Node.js | Go |
| Purpose | Serves the Backstage portal, catalog, auth | Your actual product APIs |
| Port | 7007 | 8080 (each) |
| Database | SQLite (local) | PostgreSQL |
| Who calls it | Your browser | Other services / clients |

Backstage **observes** the microservices — it reads their health and metadata. It does not call them directly in production (except the platform-dashboard plugin for health checks).

---

## How everything connects

```
You write code
      │
      ▼
Push to GitHub
      │
      ▼
CI pipeline runs automatically (ci.yml)
  ├─ tests pass? (≥70% coverage, runs against real Postgres)
  ├─ lint clean? (golangci-lint)
  ├─ Docker image builds?
  ├─ no critical CVEs? (Trivy scan)
  └─ K8s manifests valid? (kubeconform)
      │ all pass
      ▼
CD pipeline runs automatically (cd.yml)
  ├─ detects WHICH services actually changed (dorny/paths-filter)
  ├─ builds + pushes only those Docker images → GHCR
  ├─ updates image tags in k8s/services/*.yaml
  └─ deploys to Kubernetes (self-hosted runner)
      │
      ▼
Backstage (localhost:3000)
  ├─ Catalog    → shows all 6 services registered via catalog-info.yaml
  ├─ CI/CD tab  → shows GitHub Actions pipeline runs
  └─ Platform Dashboard → live service health + pod status
```

---

## Architecture overview

```
┌──────────────────────────────────────────────────────┐
│  Backstage (packages/app + packages/backend)         │
│  └─ platform-dashboard plugin                        │
│     ├─ ServiceHealthTable  (live health polling)     │
│     └─ K8sDeploymentPanel  (pod status, expand)      │
└─────────────────────────┬────────────────────────────┘
                          │ in-cluster HTTP
        ┌─────────────────▼─────────────────────────┐
        │           Kubernetes (production ns)        │
        │                                             │
        │  ┌───────────────┐  ┌──────────────────┐   │
        │  │device-catalog │  │  order-service   │   │
        │  └──────┬────────┘  └────────┬─────────┘   │
        │  ┌──────▼────────┐  ┌────────▼─────────┐   │
        │  │inventory-api  │  │ patient-service  │   │
        │  └──────┬────────┘  └────────┬─────────┘   │
        │  ┌──────▼────────┐  ┌────────▼─────────┐   │
        │  │therapy-data   │  │notification-svc  │   │
        │  └──────┬────────┘  └────────┬─────────┘   │
        │         └──────────┬──────────┘             │
        │              ┌─────▼──────┐                 │
        │              │ PostgreSQL │ (6 databases)    │
        │              └────────────┘                 │
        │  ┌──────────────────────────────────────┐   │
        │  │  Prometheus ServiceMonitor → /metrics │   │
        │  └──────────────────────────────────────┘   │
        └─────────────────────────────────────────────┘
```

---

## Quick start

### Prerequisites

- Node.js 20+, Yarn 1.22+
- Go 1.26+
- Docker Desktop
- `kubectl` + a local cluster (minikube, kind, or Docker Desktop Kubernetes)

### Run Backstage locally

Create a `.env.local` file at the project root:
```
GITHUB_TOKEN=your_pat_token
AUTH_GITHUB_CLIENT_ID=your_oauth_app_client_id
AUTH_GITHUB_CLIENT_SECRET=your_oauth_app_client_secret
```

Then start:
```bash
set -a && source .env.local && set +a && yarn start
```

Open [http://localhost:3000](http://localhost:3000) — sign in with GitHub to see the CI/CD tab.

### Run a single microservice locally

```bash
cd services/device-catalog-api
go run .

curl http://localhost:8080/health
curl http://localhost:8080/devices
curl http://localhost:8080/metrics
```

### Deploy to Kubernetes

```bash
kubectl apply -f k8s/namespaces.yaml
kubectl apply -f k8s/postgres/postgres.yaml
kubectl rollout status deployment/postgres -n production --timeout=120s
kubectl apply -f k8s/services/
kubectl apply -f k8s/monitoring/
```

---

## Services

| Service | Database | Key endpoints |
|---|---|---|
| device-catalog-api | `device_catalog` | `GET /devices`, `GET /devices/:sku`, `GET /devices/count` |
| order-service | `orders` | `GET /orders`, `POST /orders`, `GET /orders/:id` |
| inventory-api | `inventory` | `GET /inventory`, `GET /inventory/:sku` |
| patient-service | `patients` | `GET /patients`, `GET /patients/:id` |
| therapy-data-api | `therapy` | `GET /therapy`, `GET /therapy/compliance/:serial` |
| notification-service | `notifications` | `GET /notifications` |

Every service exposes `GET /health` and `GET /metrics`. See [services/README.md](services/README.md) for the full API reference.

---

## CI/CD quality gates

| Gate | Tool | Threshold |
|---|---|---|
| Unit + integration tests | `go test` + real Postgres | ≥ 70% coverage |
| Lint | golangci-lint v1.64 | zero errors |
| Docker build | docker/build-push-action | must succeed |
| Security scan | Trivy | no unfixed CRITICAL CVEs |
| K8s manifest validation | kubeconform | strict schema |

See [ARCHITECTURE.md](ARCHITECTURE.md#7-cicd-pipeline) for the full pipeline breakdown.

---

## Further reading

- [Architecture & Spec](ARCHITECTURE.md)
- [Service API Reference](services/README.md)
- [Backstage Plugin](plugins/platform-dashboard/README.md)
- [Backstage documentation](https://backstage.io/docs)
