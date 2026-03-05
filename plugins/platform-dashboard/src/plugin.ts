import {
  createPlugin,
  createRoutableExtension,
  createApiFactory,
  configApiRef,
  discoveryApiRef,
  fetchApiRef,
} from '@backstage/core-plugin-api';

import { rootRouteRef } from './routes';
import { platformDashboardApiRef } from './api/PlatformDashboardApi';
import { MockPlatformDashboardClient } from './api/MockPlatformDashboardClient';
import { PlatformDashboardClient } from './api/PlatformDashboardClient';

export const platformDashboardPlugin = createPlugin({
  id: 'platform-dashboard',
  routes: {
    root: rootRouteRef,
  },
  apis: [
    createApiFactory({
      api: platformDashboardApiRef,
      deps: { configApi: configApiRef, discoveryApi: discoveryApiRef, fetchApi: fetchApiRef },
      factory: ({ configApi, discoveryApi, fetchApi }) => {
        const dataSource = configApi.getOptionalString(
          'platformDashboard.dataSource',
        );

        if (dataSource === 'kubernetes' || dataSource === 'prometheus') {
          const namespacesRaw =
            configApi.getOptionalString('platformDashboard.kubernetes.namespace') ??
            'production,staging';

          const k8sNamespaces = namespacesRaw.split(',').map(s => s.trim());

          return new PlatformDashboardClient({
            discoveryApi,
            fetchApi,
            k8sNamespaces,
            k8sToken: '',
          });
        }

        // Default: mock
        return new MockPlatformDashboardClient();
      },
    }),
  ],
});

export const PlatformDashboardPage = platformDashboardPlugin.provide(
  createRoutableExtension({
    name: 'PlatformDashboardPage',
    component: () =>
      import('./components/PlatformDashboardPage/PlatformDashboardPage').then(
        m => m.PlatformDashboardPage,
      ),
    mountPoint: rootRouteRef,
  }),
);
