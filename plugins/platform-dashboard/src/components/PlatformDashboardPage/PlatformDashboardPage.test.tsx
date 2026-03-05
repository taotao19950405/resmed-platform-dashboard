import { renderWithEffects, wrapInTestApp, TestApiProvider } from '@backstage/test-utils';
import { platformDashboardApiRef } from '../../api/PlatformDashboardApi';
import { PlatformDashboardPage } from './PlatformDashboardPage';
import { screen } from '@testing-library/react';

const mockApi = {
  getServiceHealth: () => Promise.resolve([]),
  getK8sDeployments: () => Promise.resolve([]),
};

describe('PlatformDashboardPage', () => {
  it('renders the page header', async () => {
    await renderWithEffects(
      wrapInTestApp(
        <TestApiProvider apis={[[platformDashboardApiRef, mockApi]]}>
          <PlatformDashboardPage />
        </TestApiProvider>,
      ),
    );
    expect(screen.getByText('Platform Dashboard')).toBeInTheDocument();
  });
});
