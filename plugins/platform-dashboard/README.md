# platform-dashboard

A Backstage frontend plugin that shows a **Service Health Dashboard** and **Kubernetes Deployment Status** panel in a single unified view.

---

## Table of Contents

1. [Architecture](#architecture)
2. [Quick Start — Mock Mode](#2-quick-start--mock-mode)
3. [Full Setup — Real Kubernetes Data](#3-full-setup--real-kubernetes-data)
   - [Step 1: Install dependencies](#step-1-install-dependencies)
   - [Step 2: Start Docker Desktop](#step-2-start-docker-desktop)
   - [Step 3: Start a local Kubernetes cluster](#step-3-start-a-local-kubernetes-cluster)
   - [Step 4: Deploy sample services](#step-4-deploy-sample-services)
   - [Step 5: Deploy Prometheus](#step-5-deploy-prometheus)
   - [Step 6: Expose the APIs locally](#step-6-expose-the-apis-locally)
   - [Step 7: Switch the dashboard to live data](#step-7-switch-the-dashboard-to-live-data)
3. [Running Tests](#running-tests)
4. [Project Structure](#project-structure)
5. [Data Sources](#data-sources)
6. [Teardown](#teardown)

---

## Architecture

### How everything fits together

```
┌─────────────────────────────────────────────────────────────────────┐
│  Your Browser  (localhost:3000)                                      │
│                                                                      │
│   Backstage App                                                      │
│   ┌──────────────────────────────────────────────────────────────┐  │
│   │  PlatformDashboardPage                                        │  │
│   │  ┌───────────────────────┐  ┌───────────────────────────┐   │  │
│   │  │  ServiceHealthTable   │  │   K8sDeploymentPanel      │   │  │
│   │  │  (Status chips,       │  │   (One card per deploy,   │   │  │
│   │  │   error rate,         │  │    expand/collapse pods)  │   │  │
│   │  │   latency colours)    │  └───────────────────────────┘   │  │
│   │  └───────────────────────┘                                   │  │
│   │            │                            │                    │  │
│   │   useServiceHealth()          useK8sDeployments()            │  │
│   │            └──────────────┬─────────────┘                   │  │
│   │                           │                                  │  │
│   │              PlatformDashboardApi (ApiRef)                   │  │
│   │                           │                                  │  │
│   │           ┌───────────────┴────────────────┐                │  │
│   │           │  app-config.yaml               │                │  │
│   │           │  dataSource: mock              │                │  │
│   │           │           or kubernetes        │                │  │
│   │           └───────────────┬────────────────┘                │  │
│   │                           │                                  │  │
│   │      ┌────────────────────┴─────────────────────┐           │  │
│   │      │                                           │           │  │
│   │  MockPlatformDashboardClient        PlatformDashboardClient  │  │
│   │  (hardcoded data, ~400ms delay)     (real HTTP calls)        │  │
│   └──────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
                                                  │
                         ┌────────────────────────┴──────────────────┐
                         │                                            │
              ┌──────────┴──────────┐                    ┌───────────┴──────────┐
              │  kubectl proxy       │                    │  Prometheus           │
              │  localhost:8001      │                    │  localhost:9090       │
              │                      │                    │  (port-forward)       │
              │  Kubernetes REST API │                    │                       │
              └──────────┬──────────┘                    └───────────┬──────────┘
                         │                                            │
              ┌──────────┴──────────────────────────────────────────┐│
              │             minikube cluster (Docker)                ││
              │                                                       ││
              │  namespace: production          namespace: staging    ││
              │  ┌──────────────────────┐      ┌─────────────────┐  ││
              │  │ payment-api   ×3 pods│      │notification-svc │  ││
              │  │ order-service ×3 pods│      │  ×2 pods        │◄─┘│
              │  │ user-service  ×4 pods│      │search-api       │   │
              │  │ inventory-api ×2 pods│      │  ×2 pods        │   │
              │  └──────────────────────┘      └─────────────────┘   │
              │                                                        │
              │  namespace: monitoring                                 │
              │  ┌────────────────────────────────────────────────┐  │
              │  │ kube-prometheus-stack                           │  │
              │  │ (Prometheus + Grafana + Alertmanager)           │◄─┘
              │  │ scrapes all pods every 15s                      │
              │  └────────────────────────────────────────────────┘  │
              └───────────────────────────────────────────────────────┘
```

### Data flow explained

| Step | What happens |
|------|-------------|
| **1** | Browser loads Backstage at `localhost:3000` and navigates to `/platform-dashboard` |
| **2** | `PlatformDashboardPage` mounts and calls `useServiceHealth()` and `useK8sDeployments()` hooks |
| **3** | Both hooks call `useApi(platformDashboardApiRef)` to get the injected client |
| **4** | `plugin.ts` reads `app-config.yaml` — if `dataSource: mock` it injects `MockPlatformDashboardClient`, if `dataSource: kubernetes` it injects `PlatformDashboardClient` |
| **5 (mock)** | Mock client returns hardcoded data after ~400ms and the UI renders |
| **5 (live)** | Real client makes two parallel requests: K8s REST API (`localhost:8001`) for deployments + pods, Prometheus API (`localhost:9090`) for error rate + latency metrics |
| **6** | Data is merged — K8s gives pod count/status/restarts, Prometheus gives health metrics |
| **7** | UI renders: `ServiceHealthTable` shows one row per deployment, `K8sDeploymentPanel` shows one card per deployment with expandable pod list |

### What each piece is responsible for

| Piece | Responsibility |
|-------|---------------|
| **minikube** | Runs a single-node Kubernetes cluster inside Docker on your Mac |
| **kubectl proxy** | Opens a tunnel from `localhost:8001` to the K8s API server inside minikube — no auth token needed |
| **Prometheus** | Deployed inside the cluster, scrapes metrics from all pods every 15s |
| **port-forward** | Tunnels `localhost:9090` to the Prometheus pod inside the cluster |
| **PlatformDashboardClient** | The only piece of code that knows about external APIs — everything else is pure React |
| **ApiRef / createApiFactory** | Backstage's dependency injection — lets you swap mock ↔ real without touching any component |

---

## 1. Quick Start — Mock Mode

No Docker or Kubernetes needed. Data is hardcoded with a simulated ~400ms delay.

```bash
# From the monorepo root
yarn install
yarn start
```

Open http://localhost:3000 → click **"Platform"** in the sidebar → `/platform-dashboard`.

---

## 2. Full Setup — Real Kubernetes Data

### Step 1: Install dependencies

```bash
brew install minikube helm kubectl
```

Verify:

```bash
minikube version
helm version
kubectl version --client
```

---

### Step 2: Start Docker Desktop

Open **Docker Desktop** from your Applications folder and wait until the whale icon in the menu bar shows **"Docker Desktop is running"**.

---

### Step 3: Start a local Kubernetes cluster

```bash
minikube start --driver=docker --cpus=2 --memory=4g
```

Verify the cluster is up:

```bash
kubectl get nodes
# Should show: minikube   Ready   ...
```

---

### Step 4: Deploy sample services

All Kubernetes manifests are in the `k8s/` folder at the monorepo root.

```bash
# From monorepo root
kubectl apply -f k8s/namespaces.yaml
kubectl apply -f k8s/services/
```

This deploys:

| Service | Namespace | Description |
|---------|-----------|-------------|
| `payment-api` | `production` | nginx — simulates payment service |
| `order-service` | `production` | nginx — simulates order service |
| `user-service` | `production` | httpbin — simulates user service |
| `inventory-api` | `staging` | nginx — simulates inventory service |

Verify pods are running:

```bash
kubectl get pods -A
```

---

### Step 5: Deploy Prometheus

```bash
# Add the Prometheus Helm chart repo
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

# Install Prometheus into the cluster
helm install prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --create-namespace \
  --set prometheus.prometheusSpec.serviceMonitorSelectorNilUsesHelmValues=false
```

Wait for Prometheus to be ready (~2 minutes):

```bash
kubectl get pods -n monitoring
# All pods should show Running
```

---

### Step 6: Expose the APIs locally

Open **two separate terminals** and keep them running:

**Terminal A — Kubernetes API proxy:**
```bash
kubectl proxy --port=8001
```

**Terminal B — Prometheus port-forward:**
```bash
kubectl port-forward -n monitoring svc/prometheus-kube-prometheus-prometheus 9090:9090
```

Verify:
- Kubernetes API: http://localhost:8001/api/v1/namespaces
- Prometheus: http://localhost:9090

---

### Step 7: Switch the dashboard to live data

Edit `app-config.yaml` in the monorepo root:

```yaml
platformDashboard:
  dataSource: kubernetes        # change from 'mock' to 'kubernetes'
  prometheus:
    baseUrl: http://localhost:9090
  kubernetes:
    clusterUrl: http://localhost:8001
    namespace: production
    serviceAccountToken: ""     # empty when using kubectl proxy
```

Restart the Backstage app:

```bash
# Stop the running app (Ctrl+C), then:
yarn start
```

Open http://localhost:3000/platform-dashboard — the dashboard now shows **live pod and deployment data** from your cluster.

---

## Running Tests

```bash
# From the plugin directory
yarn test --watchAll=false       # run all tests once
yarn test --watchAll             # watch mode
yarn test --coverage             # with coverage report

# Run a specific test file
yarn test ServiceHealthTable --watchAll=false
yarn test K8sDeploymentPanel --watchAll=false
yarn test useServiceHealth --watchAll=false

# Type check
cd ../..   # monorepo root
yarn tsc --noEmit

# Lint
cd plugins/platform-dashboard
yarn lint
```

---

## Project Structure

```
plugins/platform-dashboard/
  src/
    api/
      PlatformDashboardApi.ts          # ApiRef interface
      PlatformDashboardClient.ts       # Real HTTP client (Prometheus + K8s)
      MockPlatformDashboardClient.ts   # Mock client (default)
    components/
      PlatformDashboardPage/           # Top-level routed page
      ServiceHealthTable/              # Health table + StatusChip
      K8sDeploymentPanel/              # K8s cards + PodList
    hooks/
      useServiceHealth.ts
      useK8sDeployments.ts
    types/
      index.ts                         # All shared TypeScript types
    plugin.ts                          # Plugin registration + ApiFactory
    index.ts                           # Public exports
  ACCEPTANCE_CRITERIA.md              # Full acceptance criteria from spec
  README.md                           # This file
```

---

## Data Sources

| `dataSource` value | What it does |
|--------------------|-------------|
| `mock` (default) | Returns hardcoded data with ~400ms simulated delay. No infrastructure needed. |
| `kubernetes` | Calls the real Kubernetes REST API for deployments and pods, and Prometheus for error rate / latency metrics. |

---

## Teardown

Stop the local cluster when done:

```bash
minikube stop       # pause the cluster (keeps state)
minikube delete     # destroy the cluster entirely
```
