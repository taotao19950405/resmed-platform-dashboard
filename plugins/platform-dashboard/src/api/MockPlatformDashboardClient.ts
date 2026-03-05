import { PlatformDashboardApi } from './PlatformDashboardApi';
import { ServiceHealth, K8sDeployment } from '../types';

const delay = (ms: number) => new Promise(resolve => setTimeout(resolve, ms));

const MOCK_SERVICE_HEALTH: ServiceHealth[] = [
  {
    id: 'payment-api-prod',
    name: 'payment-api',
    environment: 'production',
    owner: 'payments-team',
    status: 'unhealthy',
    errorRate: 8.3,
    latencyP99Ms: 1240,
    uptimePct: 97.2,
    lastChecked: new Date(Date.now() - 45_000).toISOString(),
    logsUrl: 'https://logs.example.com/payment-api',
    grafanaUrl: 'https://grafana.example.com/d/payment-api',
  },
  {
    id: 'order-service-prod',
    name: 'order-service',
    environment: 'production',
    owner: 'orders-team',
    status: 'degraded',
    errorRate: 2.1,
    latencyP99Ms: 920,
    uptimePct: 99.1,
    lastChecked: new Date(Date.now() - 30_000).toISOString(),
    grafanaUrl: 'https://grafana.example.com/d/order-service',
  },
  {
    id: 'user-service-prod',
    name: 'user-service',
    environment: 'production',
    owner: 'platform-team',
    status: 'healthy',
    errorRate: 0.2,
    latencyP99Ms: 85,
    uptimePct: 99.98,
    lastChecked: new Date(Date.now() - 15_000).toISOString(),
    logsUrl: 'https://logs.example.com/user-service',
    grafanaUrl: 'https://grafana.example.com/d/user-service',
  },
  {
    id: 'inventory-api-prod',
    name: 'inventory-api',
    environment: 'production',
    owner: 'inventory-team',
    status: 'healthy',
    errorRate: 0.05,
    latencyP99Ms: 140,
    uptimePct: 100,
    lastChecked: new Date(Date.now() - 20_000).toISOString(),
    logsUrl: 'https://logs.example.com/inventory-api',
  },
  {
    id: 'notification-svc-prod',
    name: 'notification-svc',
    environment: 'production',
    owner: 'comms-team',
    status: 'healthy',
    errorRate: 0.4,
    latencyP99Ms: 210,
    uptimePct: 99.7,
    lastChecked: new Date(Date.now() - 60_000).toISOString(),
    grafanaUrl: 'https://grafana.example.com/d/notification-svc',
  },
  {
    id: 'payment-api-staging',
    name: 'payment-api',
    environment: 'staging',
    owner: 'payments-team',
    status: 'healthy',
    errorRate: 0.0,
    latencyP99Ms: 75,
    uptimePct: 99.5,
    lastChecked: new Date(Date.now() - 90_000).toISOString(),
    logsUrl: 'https://logs.example.com/payment-api-staging',
  },
  {
    id: 'order-service-staging',
    name: 'order-service',
    environment: 'staging',
    owner: 'orders-team',
    status: 'degraded',
    errorRate: 3.7,
    latencyP99Ms: 650,
    uptimePct: 98.3,
    lastChecked: new Date(Date.now() - 120_000).toISOString(),
  },
  {
    id: 'search-api-staging',
    name: 'search-api',
    environment: 'staging',
    owner: 'discovery-team',
    status: 'unknown',
    errorRate: 0,
    latencyP99Ms: 0,
    uptimePct: 0,
    lastChecked: new Date(Date.now() - 600_000).toISOString(),
  },
];

const MOCK_K8S_DEPLOYMENTS: K8sDeployment[] = [
  {
    id: 'payment-api-deploy',
    serviceName: 'payment-api',
    namespace: 'production',
    version: 'v2.3.1',
    desiredPods: 3,
    readyPods: 2,
    lastDeployed: new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString(),
    imageTag: 'payment-api:v2.3.1',
    pods: [
      { name: 'payment-api-7d6f9b-xkp2m', status: 'Running', restarts: 0, age: '2h', node: 'node-1' },
      { name: 'payment-api-7d6f9b-nvq8t', status: 'Running', restarts: 1, age: '2h', node: 'node-2' },
      { name: 'payment-api-7d6f9b-rj5lw', status: 'CrashLoopBackOff', restarts: 14, age: '2h', node: 'node-3' },
    ],
  },
  {
    id: 'user-service-deploy',
    serviceName: 'user-service',
    namespace: 'production',
    version: 'v1.8.0',
    desiredPods: 4,
    readyPods: 4,
    lastDeployed: new Date(Date.now() - 5 * 24 * 60 * 60 * 1000).toISOString(),
    imageTag: 'user-service:v1.8.0',
    pods: [
      { name: 'user-service-6bc4d8-m2pqr', status: 'Running', restarts: 0, age: '5d', node: 'node-1' },
      { name: 'user-service-6bc4d8-tnx7v', status: 'Running', restarts: 0, age: '5d', node: 'node-2' },
      { name: 'user-service-6bc4d8-wk9ls', status: 'Running', restarts: 0, age: '5d', node: 'node-3' },
      { name: 'user-service-6bc4d8-fq3bz', status: 'Running', restarts: 0, age: '5d', node: 'node-4' },
    ],
  },
  {
    id: 'order-service-deploy',
    serviceName: 'order-service',
    namespace: 'production',
    version: 'v3.1.4',
    desiredPods: 3,
    readyPods: 3,
    lastDeployed: new Date(Date.now() - 1 * 24 * 60 * 60 * 1000).toISOString(),
    imageTag: 'order-service:v3.1.4',
    pods: [
      { name: 'order-service-5fd2c1-hv8qp', status: 'Running', restarts: 2, age: '1d', node: 'node-1' },
      { name: 'order-service-5fd2c1-kz4nm', status: 'Running', restarts: 0, age: '1d', node: 'node-2' },
      { name: 'order-service-5fd2c1-yw6tr', status: 'Pending', restarts: 0, age: '12m', node: 'node-3' },
    ],
  },
  {
    id: 'inventory-api-deploy',
    serviceName: 'inventory-api',
    namespace: 'production',
    version: 'v1.2.7',
    desiredPods: 2,
    readyPods: 2,
    lastDeployed: new Date(Date.now() - 10 * 24 * 60 * 60 * 1000).toISOString(),
    imageTag: 'inventory-api:v1.2.7',
    pods: [
      { name: 'inventory-api-4ae7b9-cx1pd', status: 'Running', restarts: 0, age: '10d', node: 'node-2' },
      { name: 'inventory-api-4ae7b9-qm5vn', status: 'Running', restarts: 0, age: '10d', node: 'node-4' },
    ],
  },
];

export class MockPlatformDashboardClient implements PlatformDashboardApi {
  async getServiceHealth(): Promise<ServiceHealth[]> {
    await delay(300 + Math.random() * 300);
    return MOCK_SERVICE_HEALTH;
  }

  async getK8sDeployments(): Promise<K8sDeployment[]> {
    await delay(300 + Math.random() * 300);
    return MOCK_K8S_DEPLOYMENTS;
  }
}
