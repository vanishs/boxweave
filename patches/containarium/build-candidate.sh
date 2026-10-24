#!/usr/bin/env bash
# Build an already patched source tree; never install or change services.
set -euo pipefail
if [[ $# != 2 ]]; then
  echo "Usage: $0 /absolute/patched-containarium /absolute/output" >&2
  exit 2
fi
source_root=$(realpath "$1")
output_root=$(realpath -m "$2")
[[ "$1" = /* && "$2" = /* ]] || { echo "Absolute paths required" >&2; exit 2; }
cd "$source_root"
[[ "$(git rev-parse HEAD)" = 8107ea2724dd12786bbd73ff5371e8314f3d35d9 ]] || {
  echo "Exact v0.114.2 base required" >&2; exit 2;
}
grep -q 'boxweave.tenant-network/v1' internal/server/tenant_access_server.go
mkdir -p "$output_root"
version=0.114.2-boxweave.tenant.1
flags="-X main.Version=$version -X github.com/footprintai/containarium/pkg/version.Version=$version -X github.com/footprintai/containarium/pkg/version.GitCommit=8107ea2+tenant-network"
timeout 600 go build -p 2 -ldflags "$flags" -o "$output_root/containariumd" cmd/containariumd/main.go
timeout 600 env CGO_ENABLED=0 go build -p 2 -tags containarium_client -ldflags "$flags" -o "$output_root/containarium" cmd/containarium/main.go
timeout 600 env CGO_ENABLED=0 go build -p 2 -ldflags "$flags" -o "$output_root/containarium-mcp" cmd/mcp-server/main.go
(cd "$output_root" && sha256sum containariumd containarium containarium-mcp > SHA256SUMS)
cat "$output_root/SHA256SUMS"
