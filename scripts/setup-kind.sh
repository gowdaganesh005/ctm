#!/usr/bin/env bash
set -euo pipefail

CLUSTER_NAME="cluster-timemachine"
CONFIG_FILE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/configs/kind-config.yaml"

echo "=========================================================="
echo "  Setting up kind multi-node cluster: ${CLUSTER_NAME}"
echo "=========================================================="

if kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
    echo "Cluster '${CLUSTER_NAME}' already exists."
else
    echo "Creating kind cluster using config: ${CONFIG_FILE}"
    kind create cluster --name "${CLUSTER_NAME}" --config "${CONFIG_FILE}"
fi

kubectl cluster-info --context "kind-${CLUSTER_NAME}"
echo "Kind cluster '${CLUSTER_NAME}' is ready!"
kubectl get nodes -o wide
