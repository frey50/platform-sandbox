#!/usr/bin/env bash
set -euo pipefail

CLUSTER_NAME="platform-sandbox"

echo "==> Checking prerequisites..."
command -v kind >/dev/null 2>&1 || { echo "ERROR: kind is not installed. Install it: https://kind.sigs.k8s.io/docs/user/quick-start/#installation"; exit 1; }
command -v kubectl >/dev/null 2>&1 || { echo "ERROR: kubectl is not installed."; exit 1; }
command -v docker >/dev/null 2>&1 || { echo "ERROR: docker is not installed or not running."; exit 1; }

echo "==> Checking if cluster '${CLUSTER_NAME}' already exists..."
if kind get clusters 2>/dev/null | grep -q "^${CLUSTER_NAME}$"; then
  echo "Cluster '${CLUSTER_NAME}' already exists. Skipping creation."
else
  echo "==> Creating kind cluster: ${CLUSTER_NAME} (1 control-plane, 2 workers)..."
  cat <<EOC | kind create cluster --name "${CLUSTER_NAME}" --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
  - role: control-plane
  - role: worker
  - role: worker
EOC
fi

echo "==> Setting kubectl context..."
kubectl cluster-info --context "kind-${CLUSTER_NAME}"

echo "==> Nodes:"
kubectl get nodes -o wide

echo "==> Done. Cluster '${CLUSTER_NAME}' is up."
