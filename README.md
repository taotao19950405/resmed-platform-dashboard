# Backstage — Platform Dashboard

A Backstage developer portal with a built-in **Service Health & Kubernetes Deployment Dashboard** plugin.

## Prerequisites

| Tool | Purpose | Install |
|------|---------|---------|
| Node.js 18+ | Runtime | [nodejs.org](https://nodejs.org) |
| Yarn 1.x | Package manager | `npm install -g yarn` |
| Docker Desktop | Container runtime (required for K8s mode) | [docker.com](https://www.docker.com/products/docker-desktop) |
| minikube | Local Kubernetes cluster | `brew install minikube` |
| helm | Kubernetes package manager (for Prometheus) | `brew install helm` |
| kubectl | Kubernetes CLI | `brew install kubectl` |

> **Mock mode** (default) only requires Node.js and Yarn — no Docker or Kubernetes needed.

---

## Quick Start (Mock Data)

```bash
yarn install
yarn start
```

Open http://localhost:3000 → click **"Platform"** in the sidebar.

---

## Full Setup (Real Kubernetes Data)

See [`plugins/platform-dashboard/README.md`](./plugins/platform-dashboard/README.md) for the complete guide.

---

## Running Tests

```bash
# All tests
yarn test

# Plugin tests only
cd plugins/platform-dashboard
yarn test --watchAll=false

# Type check
yarn tsc --noEmit

# Lint
yarn lint
```
