import { renderHook, act } from '@testing-library/react';
import { useServiceHealth } from './useServiceHealth';
import { platformDashboardApiRef } from '../api/PlatformDashboardApi';
import { ServiceHealth } from '../types';
import { TestApiProvider } from '@backstage/test-utils';

const MOCK_DATA: ServiceHealth[] = [
  {
    id: 'x',
    name: 'x-api',
    environment: 'production',
    owner: 'team-x',
    status: 'healthy',
    errorRate: 0,
    latencyP99Ms: 50,
    uptimePct: 100,
    lastChecked: new Date().toISOString(),
  },
];

function makeWrapper(apiImpl: any) {
  return ({ children }: { children: React.ReactNode }) => (
    <TestApiProvider apis={[[platformDashboardApiRef, apiImpl]]}>
      {children}
    </TestApiProvider>
  );
}

describe('useServiceHealth', () => {
  it('returns loading=true initially then data', async () => {
    let resolve!: (v: ServiceHealth[]) => void;
    const promise = new Promise<ServiceHealth[]>(res => {
      resolve = res;
    });
    const api = {
      getServiceHealth: () => promise,
      getK8sDeployments: () => Promise.resolve([]),
    };

    const { result } = renderHook(() => useServiceHealth(), {
      wrapper: makeWrapper(api),
    });

    expect(result.current.loading).toBe(true);
    expect(result.current.items).toEqual([]);

    await act(async () => {
      resolve(MOCK_DATA);
      await promise;
    });

    expect(result.current.loading).toBe(false);
    expect(result.current.items).toEqual(MOCK_DATA);
    expect(result.current.error).toBeUndefined();
  });

  it('returns error when API rejects', async () => {
    const err = new Error('network error');
    const api = {
      getServiceHealth: () => Promise.reject(err),
      getK8sDeployments: () => Promise.resolve([]),
    };

    const { result } = renderHook(() => useServiceHealth(), {
      wrapper: makeWrapper(api),
    });

    await act(async () => {});

    expect(result.current.loading).toBe(false);
    expect(result.current.error).toBe(err);
    expect(result.current.items).toEqual([]);
  });
});
