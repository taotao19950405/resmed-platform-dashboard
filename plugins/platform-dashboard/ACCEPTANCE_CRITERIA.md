# Acceptance Criteria
## platform-dashboard Plugin — Service Health & Kubernetes Deployment Dashboard
> Derived from: `backstage-plugin-spec.docx` v1.0 · March 2026

---

## 1. Build & Quality Gates

| # | Criterion | How to Verify |
|---|-----------|---------------|
| 1.1 | `yarn tsc --noEmit` passes with **zero errors** (strict mode) | Run from monorepo root |
| 1.2 | `yarn lint` passes with **zero warnings** | Run from plugin root |
| 1.3 | `yarn test` passes — **all tests green** | Run from plugin root |

---

## 2. Routing & Navigation

| # | Criterion | How to Verify |
|---|-----------|---------------|
| 2.1 | Plugin renders at `/platform-dashboard` in the running app | Navigate to `http://localhost:3000/platform-dashboard` |
| 2.2 | A **"Platform"** sidebar link with a dashboard icon is visible in the left nav | Launch app, inspect sidebar |
| 2.3 | Clicking the sidebar link navigates to the dashboard page | Click and confirm URL change |
| 2.4 | `app-config.yaml` contains `platformDashboard.dataSource: mock` | Inspect `app-config.yaml` |

---

## 3. Loading & Error States

| # | Criterion | How to Verify |
|---|-----------|---------------|
| 3.1 | A **loading skeleton** is displayed while data is being fetched | Observe page on first load (mock delay is 300–600 ms) |
| 3.2 | Skeleton disappears once data has loaded | Wait for data; skeleton must not persist |
| 3.3 | An **error banner** (`ErrorPanel`) is shown when the API throws | Temporarily make the mock client throw; reload page |

---

## 4. Service Health Table

| # | Criterion | How to Verify |
|---|-----------|---------------|
| 4.1 | Table shows **6 or more services** | Count rows in the rendered table |
| 4.2 | Services span at least **two environments** (production, staging) | Inspect Environment column |
| 4.3 | **Service name** column links to the Backstage catalog entity page | Inspect anchor href |
| 4.4 | **Environment** column renders a badge: blue for `production`, grey for `staging` | Visual inspection |
| 4.5 | **Status** column renders a `StatusChip` for every row | Inspect rendered chips |
| 4.6 | `StatusChip`: `healthy` → green chip with checkmark icon | Find a healthy row |
| 4.7 | `StatusChip`: `degraded` → amber/yellow chip with warning icon | Find a degraded row |
| 4.8 | `StatusChip`: `unhealthy` → red chip with error icon | Find an unhealthy row |
| 4.9 | `StatusChip`: `unknown` → grey chip with help icon | Find an unknown row |
| 4.10 | **Error Rate** column: green when `< 1%`, yellow when `1–5%`, red when `> 5%` | Compare colours to values |
| 4.11 | **P99 Latency** column: green `< 200 ms`, yellow `200–800 ms`, red `> 800 ms` | Compare colours to values |
| 4.12 | **Uptime** column shows a percentage string | Inspect cell values |
| 4.13 | **Links** column shows log icon button when `logsUrl` is present | Find a row with a logs URL |
| 4.14 | **Links** column shows Grafana icon button when `grafanaUrl` is present | Find a row with a Grafana URL |
| 4.15 | Icon buttons are **hidden** when the corresponding URL is absent | Find a row without URLs |
| 4.16 | Table columns are **sortable** (click header to toggle asc/desc) | Click a column header |

---

## 5. Mock Data — Service Health

| # | Criterion | How to Verify |
|---|-----------|---------------|
| 5.1 | At least **1 service** has `status = 'unhealthy'` and `errorRate > 5%` | Inspect table |
| 5.2 | At least **1 service** has `status = 'degraded'` and `latencyP99Ms > 800` | Inspect table |
| 5.3 | Services exist in both **`production`** and **`staging`** environments | Inspect Environment column |
| 5.4 | Mock client simulates **300–600 ms async delay** | Observe loading skeleton duration |

---

## 6. Kubernetes Deployments Panel

| # | Criterion | How to Verify |
|---|-----------|---------------|
| 6.1 | Panel renders **one card per deployment** | Count cards vs mock data |
| 6.2 | Card header shows: **service name**, namespace badge, version tag | Inspect each card header |
| 6.3 | Card shows **pod count summary** in the format `N / N Running` | Inspect card body |
| 6.4 | Pod count summary is **colour-coded**: green = all ready, yellow = partial, red = less than half | Compare colour to ready/desired ratio |
| 6.5 | Card footer shows **"Last deployed"** timestamp | Inspect card footer |
| 6.6 | Pod list is **collapsed by default** (not visible on load) | Load page; pod rows must be hidden |
| 6.7 | Clicking the expand button **shows the pod list** | Click expand arrow; rows appear |
| 6.8 | Clicking expand again **hides the pod list** | Click again; rows disappear |
| 6.9 | Each pod row shows: **name, status badge, restart count, age, node** | Expand a card and inspect rows |

---

## 7. Mock Data — Kubernetes

| # | Criterion | How to Verify |
|---|-----------|---------------|
| 7.1 | At least **1 deployment** has a pod with `status = 'CrashLoopBackOff'` | Expand the `payment-api` card |
| 7.2 | `CrashLoopBackOff` pod renders a **visually distinct red badge** | Inspect pod status chip colour |
| 7.3 | At least **1 deployment** is fully healthy: all pods `Running`, `restarts = 0` | Expand the `user-service` card |

---

## 8. TypeScript Types

| # | Criterion | How to Verify |
|---|-----------|---------------|
| 8.1 | `HealthStatus` union type defined: `'healthy' \| 'degraded' \| 'unhealthy' \| 'unknown'` | Inspect `types/index.ts` |
| 8.2 | `ServiceHealth` interface matches spec §4.1 exactly | Inspect `types/index.ts` |
| 8.3 | `PodStatus` union type defined: `'Running' \| 'Pending' \| 'CrashLoopBackOff' \| 'Terminating' \| 'Unknown'` | Inspect `types/index.ts` |
| 8.4 | `Pod` and `K8sDeployment` interfaces match spec §4.2 exactly | Inspect `types/index.ts` |

---

## 9. API & Dependency Injection

| # | Criterion | How to Verify |
|---|-----------|---------------|
| 9.1 | `platformDashboardApiRef` created with id `'plugin.platform-dashboard.service'` | Inspect `api/PlatformDashboardApi.ts` |
| 9.2 | `MockPlatformDashboardClient` registered as default via `createApiFactory` in `plugin.ts` | Inspect `plugin.ts` |
| 9.3 | Components consume the API via `useApi(platformDashboardApiRef)` — no direct imports of mock | Inspect hook files |

---

## 10. Unit Tests

| # | Criterion | Test File |
|---|-----------|-----------|
| 10.1 | `ServiceHealthTable` renders correct number of rows from mock data | `ServiceHealthTable.test.tsx` |
| 10.2 | `StatusChip` renders correct colour class per status (`healthy`, `degraded`, `unhealthy`) | `ServiceHealthTable.test.tsx` |
| 10.3 | `K8sDeploymentPanel` shows pod count summary correctly | `K8sDeploymentPanel.test.tsx` |
| 10.4 | Pod list is **hidden** by default | `K8sDeploymentPanel.test.tsx` |
| 10.5 | Pod list is **visible** after expand click | `K8sDeploymentPanel.test.tsx` |
| 10.6 | `CrashLoopBackOff` badge renders after expand | `K8sDeploymentPanel.test.tsx` |
| 10.7 | `useServiceHealth` returns `loading = true` initially, then data | `useServiceHealth.test.tsx` |
| 10.8 | `useServiceHealth` returns `error` when API rejects | `useServiceHealth.test.tsx` |
| 10.9 | `PlatformDashboardPage` renders page header | `PlatformDashboardPage.test.tsx` |

---

## 11. File Structure

All files listed in spec §3 must exist:

```
plugins/platform-dashboard/src/
  api/
    PlatformDashboardApi.ts
    PlatformDashboardClient.ts
    MockPlatformDashboardClient.ts
  components/
    PlatformDashboardPage/
      PlatformDashboardPage.tsx
      PlatformDashboardPage.test.tsx
    ServiceHealthTable/
      ServiceHealthTable.tsx
      ServiceHealthTable.test.tsx
      StatusChip.tsx
    K8sDeploymentPanel/
      K8sDeploymentPanel.tsx
      K8sDeploymentPanel.test.tsx
      PodList.tsx
  hooks/
    useServiceHealth.ts
    useK8sDeployments.ts
  types/
    index.ts
  plugin.ts
  index.ts
```

---

## 12. Non-Goals (Out of Scope for v1)

The following are explicitly **not** acceptance criteria:

- Real-time streaming (polling every 30 s is future work)
- Write / mutate operations
- Multi-cluster Kubernetes support
- Real Prometheus or Kubernetes API integration (mock is sufficient for v1)
