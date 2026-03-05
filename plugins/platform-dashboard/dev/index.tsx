import { createDevApp } from '@backstage/dev-utils';
import { platformDashboardPlugin, PlatformDashboardPage } from '../src/plugin';

createDevApp()
  .registerPlugin(platformDashboardPlugin)
  .addPage({
    element: <PlatformDashboardPage />,
    title: 'Root Page',
    path: '/platform-dashboard',
  })
  .render();
