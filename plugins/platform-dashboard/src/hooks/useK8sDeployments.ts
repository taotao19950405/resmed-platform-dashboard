import { useEffect, useState } from 'react';
import { useApi } from '@backstage/core-plugin-api';
import { platformDashboardApiRef } from '../api/PlatformDashboardApi';
import { K8sDeployment } from '../types';

export interface UseK8sDeploymentsResult {
  items: K8sDeployment[];
  loading: boolean;
  error: Error | undefined;
}

export function useK8sDeployments(): UseK8sDeploymentsResult {
  const api = useApi(platformDashboardApiRef);
  const [items, setItems] = useState<K8sDeployment[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | undefined>(undefined);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(undefined);

    api
      .getK8sDeployments()
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
