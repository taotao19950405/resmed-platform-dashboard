import { render, screen } from '@testing-library/react';
import { ServiceHealthTable } from './ServiceHealthTable';
import { ServiceHealth } from '../../types';

// Backstage Link uses react-router, so mock it simply
jest.mock('@backstage/core-components', () => {
  const actual = jest.requireActual('@backstage/core-components');
  return {
    ...actual,
    Link: ({ children }: { children: React.ReactNode }) => <span>{children}</span>,
    Table: ({ data, columns }: any) => (
      <table>
        <tbody>
          {data.map((row: any) => (
            <tr key={row.id}>
              {columns.map((col: any, i: number) => (
                <td key={i}>{col.render ? col.render(row) : row[col.field]}</td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    ),
  };
});

const MOCK_SERVICES: ServiceHealth[] = [
  {
    id: 'svc-a',
    name: 'alpha-api',
    environment: 'production',
    owner: 'team-a',
    status: 'healthy',
    errorRate: 0.1,
    latencyP99Ms: 100,
    uptimePct: 99.9,
    lastChecked: new Date().toISOString(),
  },
  {
    id: 'svc-b',
    name: 'beta-api',
    environment: 'staging',
    owner: 'team-b',
    status: 'unhealthy',
    errorRate: 7.5,
    latencyP99Ms: 1500,
    uptimePct: 95.0,
    lastChecked: new Date().toISOString(),
  },
  {
    id: 'svc-c',
    name: 'gamma-api',
    environment: 'production',
    owner: 'team-c',
    status: 'degraded',
    errorRate: 3.0,
    latencyP99Ms: 500,
    uptimePct: 98.0,
    lastChecked: new Date().toISOString(),
  },
];

describe('ServiceHealthTable', () => {
  it('renders the correct number of rows', () => {
    render(<ServiceHealthTable services={MOCK_SERVICES} />);
    expect(screen.getAllByRole('row')).toHaveLength(MOCK_SERVICES.length);
  });

  it('renders healthy StatusChip for healthy service', () => {
    render(<ServiceHealthTable services={MOCK_SERVICES} />);
    expect(screen.getByTestId('status-chip-healthy')).toBeInTheDocument();
  });

  it('renders unhealthy StatusChip for unhealthy service', () => {
    render(<ServiceHealthTable services={MOCK_SERVICES} />);
    expect(screen.getByTestId('status-chip-unhealthy')).toBeInTheDocument();
  });

  it('renders degraded StatusChip for degraded service', () => {
    render(<ServiceHealthTable services={MOCK_SERVICES} />);
    expect(screen.getByTestId('status-chip-degraded')).toBeInTheDocument();
  });

  it('shows service names', () => {
    render(<ServiceHealthTable services={MOCK_SERVICES} />);
    expect(screen.getByText('alpha-api')).toBeInTheDocument();
    expect(screen.getByText('beta-api')).toBeInTheDocument();
    expect(screen.getByText('gamma-api')).toBeInTheDocument();
  });
});
