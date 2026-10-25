#!/usr/bin/env bash
set -euo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${DIR}"

echo "Building frontend static assets..."
(cd web && npm run build)

echo "Building Go backend..."
go build -o ctm ./cmd/ctm

echo "Starting Cluster Time Machine..."
./ctm "$@"
