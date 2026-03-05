import { DiscoveryApi, FetchApi } from '@backstage/core-plugin-api';
import { PlatformDashboardApi } from './PlatformDashboardApi';
import {
  ServiceHealth,
  HealthStatus,
  K8sDeployment,
  Pod,
  PodStatus,
} from '../types';

export interface PlatformDashboardClientConfig {
  discoveryApi: DiscoveryApi;
  fetchApi: FetchApi;
  k8sNamespaces: string[];
  k8sToken?: string;
}

// ─── Kubernetes REST API response shapes ────────────────────────────────────

interface K8sPodCondition {
  type: string;
  status: string;
}

interface K8sPodContainerStatus {
  name: string;
  ready: boolean;
  restartCount: number;
  state: {
    running?: object;
    waiting?: { reason: string };
    terminated?: object;
  };
}

interface K8sPod {
  metadata: { name: string; creationTimestamp: string; namespace: string };
  spec: { nodeName?: string };
  status: {
    phase?: string;
    conditions?: K8sPodCondition[];
    containerStatuses?: K8sPodContainerStatus[];
  };
}

interface K8sDeploymentResource {
  metadata: { name: string; namespace: string; creationTimestamp: string };
  spec: { replicas: number; template: { spec: { containers: { image: string }[] } } };
  status: { readyReplicas?: number; replicas?: number };
}

interface K8sListResponse<T> {
  items: T[];
}

// ─── Prometheus response shapes ──────────────────────────────────────────────

interface PrometheusResult {
  metric: Record<string, string>;
  value: [number, string];
}

interface PrometheusQueryResponse {
  status: string;
  data: { resultType: string; result: PrometheusResult[] };
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

function podAge(creationTimestamp: string): string {
  const seconds = Math.floor(
    (Date.now() - new Date(creationTimestamp).getTime()) / 1000,
  );
  if (seconds < 60) return `${seconds}s`;
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m`;
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}h`;
  return `${Math.floor(seconds / 86400)}d`;
}

function podStatus(pod: K8sPod): PodStatus {
  const containerStatuses = pod.status.containerStatuses ?? [];
  for (const cs of containerStatuses) {
    if (cs.state?.waiting?.reason === 'CrashLoopBackOff') return 'CrashLoopBackOff';
    if (cs.state?.waiting?.reason === 'Terminating') return 'Terminating';
  }
  const phase = pod.status.phase ?? 'Unknown';
  if (phase === 'Running') return 'Running';
  if (phase === 'Pending') return 'Pending';
  if (phase === 'Terminating') return 'Terminating';
  return 'Unknown';
}

function healthFromMetrics(errorRate: number, latency: number): HealthStatus {
  if (errorRate > 5) return 'unhealthy';
  if (errorRate >= 1 || latency > 800) return 'degraded';
  return 'healthy';
}

// ─── Client ──────────────────────────────────────────────────────────────────

export class PlatformDashboardClient implements PlatformDashboardApi {
  constructor(private readonly config: PlatformDashboardClientConfig) {}

  private k8sHeaders(): HeadersInit {
    const headers: Record<string, string> = { 'Content-Type': 'application/json' };
    if (this.config.k8sToken) {
      headers.Authorization = `Bearer ${this.config.k8sToken}`;
    }
    return headers;
  }

  private async k8sFetch<T>(path: string): Promise<T> {
    const proxyBase = await this.config.discoveryApi.getBaseUrl('proxy');
    const url = `${proxyBase}/k8s${path}`;
    const res = await this.config.fetchApi.fetch(url, { headers: this.k8sHeaders(), cache: 'no-store' });
    if (!res.ok) throw new Error(`K8s API error ${res.status} for ${path}`);
    return res.json() as Promise<T>;
  }

  private async promQuery(query: string): Promise<PrometheusResult[]> {
    const proxyBase = await this.config.discoveryApi.getBaseUrl('proxy');
    const url = `${proxyBase}/prometheus/api/v1/query?query=${encodeURIComponent(query)}`;
    const res = await this.config.fetchApi.fetch(url, { cache: 'no-store' });
    if (!res.ok) return [];
    const json = (await res.json()) as PrometheusQueryResponse;
    if (json.status !== 'success') return [];
    return json.data.result;
  }

  async getServiceHealth(): Promise<ServiceHealth[]> {
    // Fetch deployments and pod metrics across all namespaces in parallel
    const [deploymentsByNs, errorRates, latencies] = await Promise.all([
      Promise.all(
        this.config.k8sNamespaces.map(ns =>
          this.k8sFetch<K8sListResponse<K8sDeploymentResource>>(
            `/apis/apps/v1/namespaces/${ns}/deployments`,
          ).then(r => ({ ns, items: r.items })),
        ),
      ),
      this.promQuery(
        'sum by (deployment, namespace) (rate(http_requests_total{status=~"5.."}[5m])) / sum by (deployment, namespace) (rate(http_requests_total[5m])) * 100',
      ),
      this.promQuery(
        'histogram_quantile(0.99, sum by (deployment, namespace, le) (rate(http_request_duration_seconds_bucket[5m]))) * 1000',
      ),
    ]);

    // Index Prometheus results by "namespace/deployment"
    const errorRateMap = new Map<string, number>();
    for (const r of errorRates) {
      const key = `${r.metric.namespace}/${r.metric.deployment}`;
      errorRateMap.set(key, parseFloat(r.value[1]));
    }

    const latencyMap = new Map<string, number>();
    for (const r of latencies) {
      const key = `${r.metric.namespace}/${r.metric.deployment}`;
      latencyMap.set(key, parseFloat(r.value[1]));
    }

    const services: ServiceHealth[] = [];

    for (const { ns, items } of deploymentsByNs) {
      for (const d of items) {
        const key = `${ns}/${d.metadata.name}`;
        const errorRate = errorRateMap.get(key) ?? 0;
        const latency = latencyMap.get(key) ?? 0;
        const readyPods = d.status.readyReplicas ?? 0;
        const desiredPods = d.spec.replicas ?? 0;

        // If no pods are ready, override status to unhealthy
        let status = healthFromMetrics(errorRate, latency);
        if (desiredPods > 0 && readyPods === 0) status = 'unhealthy';
        else if (desiredPods > 0 && readyPods < desiredPods && status === 'healthy') {
          status = 'degraded';
        }

        const uptimePct =
          desiredPods > 0
            ? parseFloat(((readyPods / desiredPods) * 100).toFixed(2))
            : 0;

        services.push({
          id: `${d.metadata.name}-${ns}`,
          name: d.metadata.name,
          environment: ns,
          owner: 'platform-team',
          status,
          errorRate: parseFloat(errorRate.toFixed(2)),
          latencyP99Ms: parseFloat(latency.toFixed(0)),
          uptimePct,
          lastChecked: new Date().toISOString(),
        });
      }
    }

    return services;
  }

  async getK8sDeployments(): Promise<K8sDeployment[]> {
    const deploymentsByNs = await Promise.all(
      this.config.k8sNamespaces.map(ns =>
        Promise.all([
          this.k8sFetch<K8sListResponse<K8sDeploymentResource>>(
            `/apis/apps/v1/namespaces/${ns}/deployments`,
          ),
          this.k8sFetch<K8sListResponse<K8sPod>>(
            `/api/v1/namespaces/${ns}/pods`,
          ),
        ]).then(([deps, pods]) => ({ ns, deps: deps.items, pods: pods.items })),
      ),
    );

    const deployments: K8sDeployment[] = [];

    for (const { ns, deps, pods } of deploymentsByNs) {
      for (const d of deps) {
        const deploymentPods = pods.filter(
          p =>
            p.metadata.name.startsWith(d.metadata.name) &&
            p.metadata.namespace === ns,
        );

        const podRows: Pod[] = deploymentPods.map(p => ({
          name: p.metadata.name,
          status: podStatus(p),
          restarts: (p.status.containerStatuses ?? []).reduce(
            (sum, cs) => sum + cs.restartCount,
            0,
          ),
          age: podAge(p.metadata.creationTimestamp),
          node: p.spec.nodeName ?? 'unknown',
        }));

        const imageTag =
          d.spec.template.spec.containers[0]?.image ?? 'unknown';

        const readyPods = podRows.filter(p => p.status === 'Running').length;

        deployments.push({
          id: `${d.metadata.name}-${ns}`,
          serviceName: d.metadata.name,
          namespace: ns,
          version: imageTag.includes(':') ? imageTag.split(':')[1] : 'latest',
          desiredPods: d.spec.replicas ?? 0,
          readyPods,
          pods: podRows,
          lastDeployed: d.metadata.creationTimestamp,
          imageTag,
        });
      }
    }

    return deployments;
  }
}
