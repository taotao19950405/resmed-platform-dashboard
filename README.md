# ResMed Platform Dashboard

A [Backstage](https://backstage.io) internal developer portal with a custom **Platform Dashboard** plugin, backed by six Go microservices, PostgreSQL, Kubernetes, and a full GitHub Actions CI/CD pipeline.

---

## Repository layout

```
.
├── packages/
│   ├── app/                        # Backstage frontend (React)
│   └── backend/                    # Backstage backend (Node)
├── plugins/
│   └── platform-dashboard/         # Custom Service Health & K8s Dashboard plugin
├── services/
│   ├── device-catalog-api/         # ResMed device catalogue (CRUD + stock)
│   ├── order-service/              # Customer order management
│   ├── inventory-api/              # Warehouse stock levels
│   ├── patient-service/            # Patient records & device assignments
│   ├── therapy-data-api/           # CPAP therapy sessions & compliance
│   └── notification-service/       # Async notification dispatch
├── k8s/
│   ├── namespaces.yaml             # production + staging namespaces
│   ├── postgres/                   # Single PostgreSQL deployment (6 databases)
│   ├── services/                   # Deployment + Service per microservice
│   └── monitoring/                 # Prometheus ServiceMonitor CRD
├── .github/
│   ├── workflows/
│   │   ├── ci.yml                  # Quality gates (test, lint, build, scan, k8s validate)
│   │   └── cd.yml                  # Build → push GHCR → update manifests → deploy
│   └── dependabot.yml              # Weekly Go module + Actions updates
└── .golangci.yml                   # Shared golangci-lint config
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
        │                                             │
        │  ┌──────────────────────────────────────┐  │
        │  │  Prometheus ServiceMonitor → /metrics │  │
        │  └──────────────────────────────────────┘  │
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

```bash
yarn install
yarn dev
```

Open [http://localhost:3000](http://localhost:3000) — the Platform Dashboard is in the left sidebar.

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
| device-catalog-api | `device_catalog` | `GET /devices`, `GET /devices/:sku` |
| order-service | `orders` | `GET /orders`, `POST /orders`, `GET /orders/:id` |
| inventory-api | `inventory` | `GET /inventory`, `GET /inventory/:sku` |
| patient-service | `patients` | `GET /patients`, `GET /patients/:id` |
| therapy-data-api | `therapy` | `GET /therapy`, `GET /therapy/compliance/:serial` |
| notification-service | `notifications` | `GET /notifications` |

Every service exposes `GET /health` and `GET /metrics`. See [services/README.md](services/README.md) for the full API reference.

---

## CI/CD

| Gate | Tool | Threshold |
|---|---|---|
| Unit tests + coverage | `go test` | 70% minimum |
| Lint | golangci-lint v1.64 | zero errors |
| Docker build | docker/build-push-action | must succeed |
| Security scan | Trivy | no unfixed CRITICAL CVEs |
| K8s manifest validation | kubeconform | strict schema |

See [ARCHITECTURE.md](ARCHITECTURE.md#7-cicd-pipeline) for the full pipeline description.

---

## Further reading

- [Architecture & Spec](ARCHITECTURE.md)
- [Service API Reference](services/README.md)
- [Backstage Plugin](plugins/platform-dashboard/README.md)
- [Backstage documentation](https://backstage.io/docs)
