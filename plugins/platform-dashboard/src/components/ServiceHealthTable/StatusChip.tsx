import Chip from '@material-ui/core/Chip';
import { makeStyles } from '@material-ui/core/styles';
import CheckCircleIcon from '@material-ui/icons/CheckCircle';
import WarningIcon from '@material-ui/icons/Warning';
import ErrorIcon from '@material-ui/icons/Error';
import HelpIcon from '@material-ui/icons/Help';
import { HealthStatus } from '../../types';

const useStyles = makeStyles(theme => ({
  healthy: {
    backgroundColor: theme.palette.success?.main ?? '#4caf50',
    color: '#fff',
  },
  degraded: {
    backgroundColor: '#ff9800',
    color: '#fff',
  },
  unhealthy: {
    backgroundColor: theme.palette.error.main,
    color: '#fff',
  },
  unknown: {
    backgroundColor: theme.palette.grey[500],
    color: '#fff',
  },
}));

const STATUS_CONFIG: Record<
  HealthStatus,
  { label: string; Icon: React.ElementType }
> = {
  healthy: { label: 'Healthy', Icon: CheckCircleIcon },
  degraded: { label: 'Degraded', Icon: WarningIcon },
  unhealthy: { label: 'Unhealthy', Icon: ErrorIcon },
  unknown: { label: 'Unknown', Icon: HelpIcon },
};

interface StatusChipProps {
  status: HealthStatus;
}

export const StatusChip = ({ status }: StatusChipProps) => {
  const classes = useStyles();
  const { label, Icon } = STATUS_CONFIG[status];

  return (
    <Chip
      size="small"
      label={label}
      icon={<Icon style={{ color: '#fff' }} />}
      className={classes[status]}
      data-testid={`status-chip-${status}`}
    />
  );
};
