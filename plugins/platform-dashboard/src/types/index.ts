// Section 4.1 — Service Health types
export type HealthStatus = 'healthy' | 'degraded' | 'unhealthy' | 'unknown';

export interface ServiceHealth {
  id: string;
  name: string;           // e.g. 'payment-api'
  environment: string;    // e.g. 'production', 'staging'
  owner: string;          // team name
  status: HealthStatus;
  errorRate: number;      // percentage 0–100
  latencyP99Ms: number;   // milliseconds
  uptimePct: number;      // percentage 0–100
  lastChecked: string;    // ISO 8601
  logsUrl?: string;
  grafanaUrl?: string;
}

// Section 4.2 — Kubernetes Deployment types
export type PodStatus =
  | 'Running'
  | 'Pending'
  | 'CrashLoopBackOff'
  | 'Terminating'
  | 'Unknown';

export interface Pod {
  name: string;
  status: PodStatus;
  restarts: number;
  age: string;  // human-readable, e.g. '2d'
  node: string;
}

export interface K8sDeployment {
  id: string;
  serviceName: string;
  namespace: string;
  version: string;        // e.g. 'v1.4.2'
  desiredPods: number;
  readyPods: number;
  pods: Pod[];
  lastDeployed: string;   // ISO 8601
  imageTag: string;
}
