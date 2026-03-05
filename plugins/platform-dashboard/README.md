# platform-dashboard

Backstage plugin — **Service Health & Kubernetes Deployment Dashboard** for the ResMed platform.

---

## Features

- **Service Health Table** — live health status for all backend microservices with colour-coded chips (healthy / degraded / down), sortable columns, and direct links to each service
- **K8s Deployment Panel** — expandable cards per Kubernetes deployment showing pod names, readiness state, and restart counts
- Auto-refreshes every 30 seconds via polling hooks

---

## Installation

The plugin is already wired into the Backstage app. To install in a different instance:

**1. Add the plugin dependency**
```bash
yarn workspace app add @internal/plugin-platform-dashboard
```

**2. Register the route in `packages/app/src/App.tsx`**
```tsx
import { PlatformDashboardPage } from '@internal/plugin-platform-dashboard';

// Inside <FlatRoutes>:
<Route path="/platform-dashboard" element={<PlatformDashboardPage />} />
```

**3. Add the sidebar link in `packages/app/src/components/Root/Root.tsx`**
```tsx
import DashboardIcon from '@material-ui/icons/Dashboard';
import { SidebarItem } from '@backstage/core-components';

<SidebarItem icon={DashboardIcon} to="platform-dashboard" text="Platform" />
```

---

## Configuration

```yaml
# app-config.yaml
platformDashboard:
  baseUrl: ''   # leave empty to use mock client
```

---

## API

The plugin uses `platformDashboardApiRef`. The default factory registers `MockPlatformDashboardClient` (8 services, 4 K8s deployments, 300–600ms simulated delay).

To connect a real backend, implement `PlatformDashboardApi`:

```typescript
interface PlatformDashboardApi {
  getServiceHealth(): Promise<ServiceHealth[]>;
  getK8sDeployments(): Promise<K8sDeployment[]>;
}
```

### Types

```typescript
type HealthStatus = 'healthy' | 'degraded' | 'down' | 'unknown';

interface ServiceHealth {
  name: string;
  status: HealthStatus;
  latencyMs: number;
  endpoint: string;
  version?: string;
  lastChecked: string;
}

interface K8sDeployment {
  name: string;
  namespace: string;
  replicas: number;
  readyReplicas: number;
  pods: Pod[];
}

interface Pod {
  name: string;
  status: HealthStatus;
  restarts: number;
  age: string;
}
```

---

## Development

```bash
yarn dev                                                    # run Backstage
yarn tsc --noEmit                                           # type-check
yarn workspace @internal/plugin-platform-dashboard test    # run tests
```

---

## File structure

```
src/
├── api/
│   ├── PlatformDashboardApi.ts         # Interface + ApiRef
│   ├── MockPlatformDashboardClient.ts  # Dev mock
│   └── PlatformDashboardClient.ts      # Production client stub
├── hooks/
│   ├── useServiceHealth.ts
│   └── useK8sDeployments.ts
├── components/
│   ├── StatusChip/
│   ├── ServiceHealthTable/
│   ├── K8sDeploymentPanel/
│   └── PlatformDashboardPage/
├── types/index.ts
├── plugin.ts
└── index.ts
```
