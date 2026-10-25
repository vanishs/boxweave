#!/usr/bin/env bash
# Build the reviewed native Tenant branch; never install or change services.
set -euo pipefail
if [[ $# != 2 ]]; then
  echo "Usage: $0 /absolute/containarium /absolute/output" >&2
  exit 2
fi
source_root=$(realpath "$1")
output_root=$(realpath -m "$2")
[[ "$1" = /* && "$2" = /* ]] || { echo "Absolute paths required" >&2; exit 2; }
cd "$source_root"
[[ "$(git rev-parse HEAD)" = c13582b5016e7471d83d50d1e09a49a2fe661395 ]] || {
  echo "Exact reviewed native-network-tenant commit required" >&2; exit 2;
}
[[ -z "$(git status --porcelain --untracked-files=normal)" ]] || { echo "Clean reviewed source required" >&2; exit 2; }
grep -q 'boxweave.tenant-network/v1' internal/server/network_tenant_capabilities.go
mkdir -p "$output_root"
version=0.115.0-boxweave.tenant.c13582b
flags="-X main.Version=$version -X github.com/footprintai/containarium/pkg/version.Version=$version -X github.com/footprintai/containarium/pkg/version.GitCommit=c13582b"
timeout 600 go build -p 2 -ldflags "$flags" -o "$output_root/containariumd" cmd/containariumd/main.go
timeout 600 env CGO_ENABLED=0 go build -p 2 -tags containarium_client -ldflags "$flags" -o "$output_root/containarium" cmd/containarium/main.go
timeout 600 env CGO_ENABLED=0 go build -p 2 -ldflags "$flags" -o "$output_root/containarium-mcp" cmd/mcp-server/main.go
(cd "$output_root" && sha256sum containariumd containarium containarium-mcp > SHA256SUMS)
cat "$output_root/SHA256SUMS"
