import { render, screen, fireEvent } from '@testing-library/react';
import { K8sDeploymentCard } from './K8sDeploymentPanel';
import { K8sDeployment } from '../../types';

const FULL_DEPLOY: K8sDeployment = {
  id: 'svc-healthy',
  serviceName: 'user-service',
  namespace: 'production',
  version: 'v1.0.0',
  desiredPods: 3,
  readyPods: 3,
  lastDeployed: new Date().toISOString(),
  imageTag: 'user-service:v1.0.0',
  pods: [
    { name: 'pod-a', status: 'Running', restarts: 0, age: '1d', node: 'node-1' },
    { name: 'pod-b', status: 'Running', restarts: 0, age: '1d', node: 'node-2' },
    { name: 'pod-c', status: 'Running', restarts: 0, age: '1d', node: 'node-3' },
  ],
};

const CRASH_DEPLOY: K8sDeployment = {
  id: 'svc-crash',
  serviceName: 'broken-api',
  namespace: 'production',
  version: 'v2.0.0',
  desiredPods: 2,
  readyPods: 1,
  lastDeployed: new Date().toISOString(),
  imageTag: 'broken-api:v2.0.0',
  pods: [
    { name: 'broken-pod-1', status: 'Running', restarts: 0, age: '3h', node: 'node-1' },
    { name: 'broken-pod-2', status: 'CrashLoopBackOff', restarts: 12, age: '3h', node: 'node-2' },
  ],
};

describe('K8sDeploymentCard', () => {
  it('shows pod count summary correctly', () => {
    render(<K8sDeploymentCard deployment={FULL_DEPLOY} />);
    expect(screen.getByText('3 / 3 Running')).toBeInTheDocument();
  });

  it('pod list is not in the DOM by default (collapsed)', () => {
    render(<K8sDeploymentCard deployment={FULL_DEPLOY} />);
    expect(screen.queryByTestId(`pod-list-${FULL_DEPLOY.id}`)).not.toBeInTheDocument();
  });

  it('pod list appears in the DOM after expand click', () => {
    render(<K8sDeploymentCard deployment={FULL_DEPLOY} />);
    fireEvent.click(screen.getByTestId(`expand-pods-${FULL_DEPLOY.id}`));
    expect(screen.getByTestId(`pod-list-${FULL_DEPLOY.id}`)).toBeInTheDocument();
  });

  it('shows CrashLoopBackOff badge for crashing pod', () => {
    render(<K8sDeploymentCard deployment={CRASH_DEPLOY} />);
    fireEvent.click(screen.getByTestId(`expand-pods-${CRASH_DEPLOY.id}`));
    expect(screen.getByTestId('pod-status-CrashLoopBackOff')).toBeInTheDocument();
  });

  it('shows pod count summary for partial deployment', () => {
    render(<K8sDeploymentCard deployment={CRASH_DEPLOY} />);
    expect(screen.getByText('1 / 2 Running')).toBeInTheDocument();
  });
});
