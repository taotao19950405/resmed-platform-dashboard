export interface Config {
  platformDashboard?: {
    /**
     * Data source to use: 'mock' or 'kubernetes'
     * @visibility frontend
     */
    dataSource?: string;

    prometheus?: {
      /**
       * Base URL for the Prometheus HTTP API
       * @visibility frontend
       */
      baseUrl?: string;
    };

    kubernetes?: {
      /**
       * URL for the Kubernetes API server (kubectl proxy)
       * @visibility frontend
       */
      clusterUrl?: string;

      /**
       * Comma-separated list of namespaces to watch
       * @visibility frontend
       */
      namespace?: string;

      /**
       * Service account token (leave empty when using kubectl proxy)
       * @visibility secret
       */
      serviceAccountToken?: string;
    };
  };
}
