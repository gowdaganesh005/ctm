#!/usr/bin/env bash
set -euo pipefail

echo "=========================================================="
echo "  Deploying Sample Workloads to Active Kubernetes Cluster"
echo "=========================================================="

kubectl create namespace production --dry-run=client -o yaml | kubectl apply -f -
kubectl create namespace staging --dry-run=client -o yaml | kubectl apply -f -

# 1. Production Payment Deployment
cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: payment-processor
  namespace: production
  labels:
    app: payment-processor
spec:
  replicas: 2
  selector:
    matchLabels:
      app: payment-processor
  template:
    metadata:
      labels:
        app: payment-processor
    spec:
      containers:
      - name: payment-processor
        image: redis:7.2-alpine
        resources:
          limits:
            memory: "256Mi"
            cpu: "200m"
          requests:
            memory: "128Mi"
            cpu: "100m"
        env:
        - name: DATABASE_PASSWORD
          value: "super_secret_db_pass_redacted"
        - name: API_ENV
          value: "production"
EOF

# 2. Production Secret (Sensitive values to test SHA256 hashing)
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Secret
metadata:
  name: stripe-credentials
  namespace: production
type: Opaque
stringData:
  api_key: "sk_live_very_secret_token_12345"
  webhook_secret: "whsec_abcdef0123456789"
EOF

# 3. Production Service
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Service
metadata:
  name: payment-service
  namespace: production
spec:
  selector:
    app: payment-processor
  ports:
  - protocol: TCP
    port: 80
    targetPort: 6379
EOF

echo "Sample workloads deployed successfully!"
kubectl get pods -n production
