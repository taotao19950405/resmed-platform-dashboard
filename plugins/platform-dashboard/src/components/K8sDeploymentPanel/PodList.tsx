import Table from '@material-ui/core/Table';
import TableBody from '@material-ui/core/TableBody';
import TableCell from '@material-ui/core/TableCell';
import TableHead from '@material-ui/core/TableHead';
import TableRow from '@material-ui/core/TableRow';
import Chip from '@material-ui/core/Chip';
import { makeStyles } from '@material-ui/core/styles';
import { Pod, PodStatus } from '../../types';

const useStyles = makeStyles({
  running: { backgroundColor: '#4caf50', color: '#fff' },
  pending: { backgroundColor: '#ff9800', color: '#fff' },
  crashloop: { backgroundColor: '#f44336', color: '#fff' },
  terminating: { backgroundColor: '#9e9e9e', color: '#fff' },
  unknown: { backgroundColor: '#757575', color: '#fff' },
});

function PodStatusBadge({ status }: { status: PodStatus }) {
  const classes = useStyles();
  const classMap: Record<PodStatus, string> = {
    Running: classes.running,
    Pending: classes.pending,
    CrashLoopBackOff: classes.crashloop,
    Terminating: classes.terminating,
    Unknown: classes.unknown,
  };
  return (
    <Chip
      size="small"
      label={status}
      className={classMap[status]}
      data-testid={`pod-status-${status}`}
    />
  );
}

interface PodListProps {
  pods: Pod[];
}

export const PodList = ({ pods }: PodListProps) => (
  <Table size="small" aria-label="pod list">
    <TableHead>
      <TableRow>
        <TableCell>Pod Name</TableCell>
        <TableCell>Status</TableCell>
        <TableCell align="right">Restarts</TableCell>
        <TableCell>Age</TableCell>
        <TableCell>Node</TableCell>
      </TableRow>
    </TableHead>
    <TableBody>
      {pods.map(pod => (
        <TableRow key={pod.name}>
          <TableCell>
            <code>{pod.name}</code>
          </TableCell>
          <TableCell>
            <PodStatusBadge status={pod.status} />
          </TableCell>
          <TableCell align="right">{pod.restarts}</TableCell>
          <TableCell>{pod.age}</TableCell>
          <TableCell>{pod.node}</TableCell>
        </TableRow>
      ))}
    </TableBody>
  </Table>
);
