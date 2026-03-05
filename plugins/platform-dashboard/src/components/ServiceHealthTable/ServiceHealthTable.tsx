import { useState } from 'react';
import {
  Table,
  TableColumn,
  Link,
} from '@backstage/core-components';
import Chip from '@material-ui/core/Chip';
import Tooltip from '@material-ui/core/Tooltip';
import IconButton from '@material-ui/core/IconButton';
import ListAltIcon from '@material-ui/icons/ListAlt';
import ShowChartIcon from '@material-ui/icons/ShowChart';
import TableSortLabel from '@material-ui/core/TableSortLabel';
import { makeStyles } from '@material-ui/core/styles';
import { ServiceHealth } from '../../types';
import { StatusChip } from './StatusChip';

const useStyles = makeStyles(_theme => ({
  envBadgeProd: {
    backgroundColor: '#1976d2',
    color: '#fff',
  },
  envBadgeStaging: {
    backgroundColor: '#757575',
    color: '#fff',
  },
  green: { color: '#4caf50', fontWeight: 600 },
  yellow: { color: '#ff9800', fontWeight: 600 },
  red: { color: '#f44336', fontWeight: 600 },
  linksCell: { display: 'flex', gap: 4 },
}));

function getErrorRateClass(
  value: number,
  classes: ReturnType<typeof useStyles>,
) {
  if (value > 5) return classes.red;
  if (value >= 1) return classes.yellow;
  return classes.green;
}

function getLatencyClass(
  value: number,
  classes: ReturnType<typeof useStyles>,
) {
  if (value > 800) return classes.red;
  if (value >= 200) return classes.yellow;
  return classes.green;
}

function ErrorRateCell({ value }: { value: number }) {
  const classes = useStyles();
  return (
    <span className={getErrorRateClass(value, classes)}>
      {value.toFixed(2)}%
    </span>
  );
}

function LatencyCell({ value }: { value: number }) {
  const classes = useStyles();
  return (
    <span className={getLatencyClass(value, classes)}>{value} ms</span>
  );
}

function EnvBadge({ env }: { env: string }) {
  const classes = useStyles();
  return (
    <Chip
      size="small"
      label={env}
      className={
        env === 'production' ? classes.envBadgeProd : classes.envBadgeStaging
      }
    />
  );
}

interface Props {
  services: ServiceHealth[];
}

type SortKey = keyof ServiceHealth;
type SortDir = 'asc' | 'desc';

export const ServiceHealthTable = ({ services }: Props) => {
  const classes = useStyles();
  const [sortKey, setSortKey] = useState<SortKey>('name');
  const [sortDir, setSortDir] = useState<SortDir>('asc');

  const handleSort = (key: SortKey) => {
    if (key === sortKey) {
      setSortDir(d => (d === 'asc' ? 'desc' : 'asc'));
    } else {
      setSortKey(key);
      setSortDir('asc');
    }
  };

  const sorted = [...services].sort((a, b) => {
    const av = a[sortKey];
    const bv = b[sortKey];
    let cmp = 0;
    if (av! < bv!) cmp = -1;
    else if (av! > bv!) cmp = 1;
    return sortDir === 'asc' ? cmp : -cmp;
  });

  const SortLabel = ({ col, label }: { col: SortKey; label: string }) => (
    <TableSortLabel
      active={sortKey === col}
      direction={sortKey === col ? sortDir : 'asc'}
      onClick={() => handleSort(col)}
    >
      {label}
    </TableSortLabel>
  );

  const columns: TableColumn<ServiceHealth>[] = [
    {
      title: <SortLabel col="name" label="Service" />,
      field: 'name',
      render: row => (
        <Link to={`/catalog/default/component/${row.name}`}>
          {row.name}
        </Link>
      ),
    },
    {
      title: <SortLabel col="environment" label="Environment" />,
      field: 'environment',
      render: row => <EnvBadge env={row.environment} />,
    },
    {
      title: <SortLabel col="status" label="Status" />,
      field: 'status',
      render: row => <StatusChip status={row.status} />,
    },
    {
      title: <SortLabel col="errorRate" label="Error Rate" />,
      field: 'errorRate',
      render: row => <ErrorRateCell value={row.errorRate} />,
    },
    {
      title: <SortLabel col="latencyP99Ms" label="P99 Latency" />,
      field: 'latencyP99Ms',
      render: row => <LatencyCell value={row.latencyP99Ms} />,
    },
    {
      title: <SortLabel col="uptimePct" label="Uptime" />,
      field: 'uptimePct',
      render: row => <span>{row.uptimePct.toFixed(2)}%</span>,
    },
    {
      title: 'Links',
      field: 'id',
      render: row => (
        <span className={classes.linksCell}>
          {row.logsUrl && (
            <Tooltip title="Logs">
              <IconButton size="small" href={row.logsUrl} target="_blank">
                <ListAltIcon fontSize="small" />
              </IconButton>
            </Tooltip>
          )}
          {row.grafanaUrl && (
            <Tooltip title="Grafana">
              <IconButton size="small" href={row.grafanaUrl} target="_blank">
                <ShowChartIcon fontSize="small" />
              </IconButton>
            </Tooltip>
          )}
        </span>
      ),
    },
  ];

  return (
    <Table<ServiceHealth>
      title="Service Health"
      options={{ paging: false, search: false }}
      columns={columns}
      data={sorted}
    />
  );
};
