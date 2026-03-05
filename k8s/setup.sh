#!/usr/bin/env bash
set -e

echo "==> Starting minikube cluster..."
minikube start --driver=docker --cpus=2 --memory=4g

echo "==> Creating namespaces..."
kubectl apply -f k8s/namespaces.yaml

echo "==> Deploying services..."
kubectl apply -f k8s/services/

echo "==> Adding Prometheus Helm repo..."
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

echo "==> Installing Prometheus (kube-prometheus-stack)..."
helm upgrade --install prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --create-namespace \
  --set prometheus.prometheusSpec.serviceMonitorSelectorNilUsesHelmValues=false \
  --wait

echo "==> Applying ServiceMonitor..."
kubectl apply -f k8s/monitoring/prometheus-servicemonitor.yaml

echo ""
echo "✅ All done! Cluster and services are running."
echo ""
echo "Now run these in two separate terminals:"
echo "  Terminal A:  kubectl proxy --port=8001"
echo "  Terminal B:  kubectl port-forward -n monitoring svc/prometheus-kube-prometheus-prometheus 9090:9090"
echo ""
echo "Then update app-config.yaml: dataSource: kubernetes"
echo "And restart Backstage: yarn start"
