// Section 5 — API interface and ApiRef
import { createApiRef } from '@backstage/core-plugin-api';
import { ServiceHealth, K8sDeployment } from '../types';

export interface PlatformDashboardApi {
  getServiceHealth(): Promise<ServiceHealth[]>;
  getK8sDeployments(): Promise<K8sDeployment[]>;
}

export const platformDashboardApiRef = createApiRef<PlatformDashboardApi>({
  id: 'plugin.platform-dashboard.service',
});
