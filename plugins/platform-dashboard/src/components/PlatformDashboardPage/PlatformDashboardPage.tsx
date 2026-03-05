import {
  Header,
  Page,
  Content,
  ContentHeader,
  SupportButton,
  ErrorPanel,
} from '@backstage/core-components';
import Grid from '@material-ui/core/Grid';
import Skeleton from '@material-ui/lab/Skeleton';
import Typography from '@material-ui/core/Typography';
import { useServiceHealth } from '../../hooks/useServiceHealth';
import { useK8sDeployments } from '../../hooks/useK8sDeployments';
import { ServiceHealthTable } from '../ServiceHealthTable/ServiceHealthTable';
import { K8sDeploymentPanel } from '../K8sDeploymentPanel/K8sDeploymentPanel';

function LoadingSkeleton() {
  return (
    <Grid container spacing={3} direction="column">
      <Grid item>
        <Skeleton variant="rect" height={40} />
      </Grid>
      <Grid item>
        <Skeleton variant="rect" height={200} />
      </Grid>
      <Grid item>
        <Skeleton variant="rect" height={40} />
      </Grid>
      <Grid item>
        <Skeleton variant="rect" height={200} />
      </Grid>
    </Grid>
  );
}

export const PlatformDashboardPage = () => {
  const health = useServiceHealth();
  const k8s = useK8sDeployments();

  const loading = health.loading || k8s.loading;
  const error = health.error ?? k8s.error;

  return (
    <Page themeId="tool">
      <Header
        title="Platform Dashboard"
        subtitle="Service Health & Kubernetes Deployments"
      />
      <Content>
        <ContentHeader title="">
          <SupportButton>
            View service health and Kubernetes deployment status across all
            environments.
          </SupportButton>
        </ContentHeader>

        {loading && <LoadingSkeleton />}

        {!loading && error && (
          <ErrorPanel error={error} title="Failed to load dashboard data" />
        )}

        {!loading && !error && (
          <Grid container spacing={4} direction="column">
            <Grid item xs={12}>
              <Typography variant="h5" gutterBottom>
                Service Health
              </Typography>
              <ServiceHealthTable services={health.items} />
            </Grid>
            <Grid item xs={12}>
              <Typography variant="h5" gutterBottom>
                Kubernetes Deployments
              </Typography>
              <K8sDeploymentPanel deployments={k8s.items} />
            </Grid>
          </Grid>
        )}
      </Content>
    </Page>
  );
};
