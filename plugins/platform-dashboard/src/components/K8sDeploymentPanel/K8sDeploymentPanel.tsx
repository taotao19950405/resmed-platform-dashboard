import { useState } from 'react';
import Card from '@material-ui/core/Card';
import CardHeader from '@material-ui/core/CardHeader';
import CardContent from '@material-ui/core/CardContent';
import CardActions from '@material-ui/core/CardActions';
import Collapse from '@material-ui/core/Collapse';
import IconButton from '@material-ui/core/IconButton';
import Chip from '@material-ui/core/Chip';
import Typography from '@material-ui/core/Typography';
import ExpandMoreIcon from '@material-ui/icons/ExpandMore';
import ExpandLessIcon from '@material-ui/icons/ExpandLess';
import { makeStyles } from '@material-ui/core/styles';
import Grid from '@material-ui/core/Grid';
import { K8sDeployment } from '../../types';
import { PodList } from './PodList';

const useStyles = makeStyles(theme => ({
  card: {
    marginBottom: theme.spacing(2),
  },
  namespaceBadge: {
    backgroundColor: '#1976d2',
    color: '#fff',
    marginLeft: theme.spacing(1),
  },
  versionBadge: {
    backgroundColor: '#7b1fa2',
    color: '#fff',
    marginLeft: theme.spacing(1),
  },
  podSummaryGreen: { color: '#4caf50', fontWeight: 600 },
  podSummaryYellow: { color: '#ff9800', fontWeight: 600 },
  podSummaryRed: { color: '#f44336', fontWeight: 600 },
  footer: {
    color: theme.palette.text.secondary,
    fontSize: '0.75rem',
    paddingTop: theme.spacing(1),
  },
  expandButton: { marginLeft: 'auto' },
}));

interface Props {
  deployment: K8sDeployment;
}

export const K8sDeploymentCard = ({ deployment }: Props) => {
  const classes = useStyles();
  const [expanded, setExpanded] = useState(false);

  const ratio = deployment.readyPods / deployment.desiredPods;
  let podSummaryClass = classes.podSummaryRed;
  if (ratio === 1) podSummaryClass = classes.podSummaryGreen;
  else if (ratio >= 0.5) podSummaryClass = classes.podSummaryYellow;

  const lastDeployedLabel = new Date(deployment.lastDeployed).toLocaleString();

  return (
    <Card className={classes.card} data-testid={`k8s-card-${deployment.id}`}>
      <CardHeader
        title={
          <span>
            {deployment.serviceName}
            <Chip
              size="small"
              label={deployment.namespace}
              className={classes.namespaceBadge}
            />
            <Chip
              size="small"
              label={deployment.version}
              className={classes.versionBadge}
            />
          </span>
        }
        subheader={deployment.imageTag}
      />
      <CardContent>
        <Typography variant="body2" className={podSummaryClass}>
          {deployment.readyPods} / {deployment.desiredPods} Running
        </Typography>
        <Typography className={classes.footer}>
          Last deployed: {lastDeployedLabel}
        </Typography>
      </CardContent>
      <CardActions disableSpacing>
        <Typography variant="caption">
          {deployment.pods.length} pod{deployment.pods.length !== 1 ? 's' : ''}
        </Typography>
        <IconButton
          className={classes.expandButton}
          onClick={() => setExpanded(e => !e)}
          aria-label={expanded ? 'collapse pod list' : 'expand pod list'}
          size="small"
          data-testid={`expand-pods-${deployment.id}`}
        >
          {expanded ? <ExpandLessIcon /> : <ExpandMoreIcon />}
        </IconButton>
      </CardActions>
      <Collapse in={expanded} unmountOnExit>
        <CardContent data-testid={`pod-list-${deployment.id}`}>
          <PodList pods={deployment.pods} />
        </CardContent>
      </Collapse>
    </Card>
  );
};

interface PanelProps {
  deployments: K8sDeployment[];
}

export const K8sDeploymentPanel = ({ deployments }: PanelProps) => (
  <Grid container spacing={2}>
    {deployments.map(d => (
      <Grid item xs={12} md={6} key={d.id}>
        <K8sDeploymentCard deployment={d} />
      </Grid>
    ))}
  </Grid>
);
