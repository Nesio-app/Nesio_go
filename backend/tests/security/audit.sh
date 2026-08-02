#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/../.."

echo "[1/2] go vuln scan"
if command -v govulncheck >/dev/null 2>&1; then
  govulncheck ./...
else
  echo "govulncheck not installed; install with: go install golang.org/x/vuln/cmd/govulncheck@latest"
fi

echo "[2/2] gosec scan"
if command -v gosec >/dev/null 2>&1; then
  gosec ./...
else
  echo "gosec not installed; install with: go install github.com/securego/gosec/v2/cmd/gosec@latest"
fi
