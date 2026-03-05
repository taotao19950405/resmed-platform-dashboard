import { useEffect, useState } from 'react';
import { useApi } from '@backstage/core-plugin-api';
import { platformDashboardApiRef } from '../api/PlatformDashboardApi';
import { ServiceHealth } from '../types';

export interface UseServiceHealthResult {
  items: ServiceHealth[];
  loading: boolean;
  error: Error | undefined;
}

export function useServiceHealth(): UseServiceHealthResult {
  const api = useApi(platformDashboardApiRef);
  const [items, setItems] = useState<ServiceHealth[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | undefined>(undefined);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(undefined);

    api
      .getServiceHealth()
      .then(data => {
        if (!cancelled) {
          setItems(data);
          setLoading(false);
        }
      })
      .catch(err => {
        if (!cancelled) {
          setError(err);
          setLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [api]);

  return { items, loading, error };
}
