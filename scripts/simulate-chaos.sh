#!/usr/bin/env bash
set -euo pipefail

SCENARIO="${1:-oom}"

echo "=========================================================="
echo "  Executing Real Kubernetes Chaos Scenario: ${SCENARIO}"
echo "=========================================================="

case "${SCENARIO}" in
  oom)
    echo "Simulating Human 'kubectl patch' memory limit squeeze -> OOMKilled..."
    kubectl patch deployment payment-processor -n production --type='json' -p='[
      {"op": "replace", "path": "/spec/template/spec/containers/0/resources/limits/memory", "value": "12Mi"},
      {"op": "replace", "path": "/spec/template/spec/containers/0/resources/requests/memory", "value": "12Mi"}
    ]'
    echo "Deployment patched! Check Cluster Time Machine for REVISION_ADDED and OOMKilled alert."
    ;;
  image)
    echo "Simulating Human 'kubectl set image' with invalid tag -> ImagePullBackOff..."
    kubectl set image deployment/payment-processor payment-processor=redis:invalid-tag-xyz999 -n production
    echo "Image updated! Check Cluster Time Machine for ImagePullBackOff alert."
    ;;
  cordon)
    echo "Simulating Node cordon/drain..."
    WORKER=$(kubectl get nodes -l node-role.kubernetes.io/control-plane!="" -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || kubectl get nodes -o jsonpath='{.items[1].metadata.name}')
    echo "Cordoning node: ${WORKER}"
    kubectl cordon "${WORKER}"
    echo "Node cordoned! Check Cluster Time Machine for Node revision."
    ;;
  restore)
    echo "Restoring payment-processor limits and uncordoning nodes..."
    kubectl patch deployment payment-processor -n production --type='json' -p='[
      {"op": "replace", "path": "/spec/template/spec/containers/0/resources/limits/memory", "value": "256Mi"},
      {"op": "replace", "path": "/spec/template/spec/containers/0/resources/requests/memory", "value": "128Mi"}
    ]' || true
    kubectl set image deployment/payment-processor payment-processor=redis:7.2-alpine -n production || true
    kubectl uncordon $(kubectl get nodes -o jsonpath='{.items[*].metadata.name}') || true
    echo "Cluster restored to healthy baseline."
    ;;
  *)
    echo "Usage: $0 [oom|image|cordon|restore]"
    exit 1
    ;;
esac
