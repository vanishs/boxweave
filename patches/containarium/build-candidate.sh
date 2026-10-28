#!/usr/bin/env bash
# Build the pinned upstream source profile; never install or change services.
set -euo pipefail
if [[ $# != 2 ]]; then
  echo "Usage: $0 /absolute/containarium /absolute/output" >&2
  exit 2
fi
profile_path="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/native-network-tenant.json"
profile_values=$(python3 - "$profile_path" <<'PY'
import json,re,sys
profile=json.load(open(sys.argv[1],encoding="utf-8"))
if not re.fullmatch(r"[0-9a-f]{40}",profile["commit"]):
    raise SystemExit("Invalid pinned upstream commit")
if profile["contract"] != "native-network-tenant/v1":
    raise SystemExit("Unrecognized native Tenant contract")
print(profile["commit"])
print(profile["contract"])
PY
)
mapfile -t profile <<< "$profile_values"
commit=${profile[0]}
contract=${profile[1]}
source_root=$(realpath "$1")
output_root=$(realpath -m "$2")
[[ "$1" = /* && "$2" = /* ]] || { echo "Absolute paths required" >&2; exit 2; }
cd "$source_root"
[[ "$(git rev-parse HEAD)" = "$commit" ]] || {
  echo "Exact reviewed upstream commit required ($commit)" >&2; exit 2;
}
[[ -z "$(git status --porcelain --untracked-files=normal)" ]] || { echo "Clean reviewed source required" >&2; exit 2; }
grep -Fq "const NativeTenantContractID = \"$contract\"" pkg/core/incus/network_tenant.go
mkdir -p "$output_root"
# Source builds are not an official release and must not claim its version.
version="0.0.0-source.${commit:0:12}"
flags="-X github.com/footprintai/containarium/pkg/version.Version=$version -X github.com/footprintai/containarium/pkg/version.GitCommit=$commit"
timeout 600 go build -p 2 -ldflags "$flags" -o "$output_root/containariumd" ./cmd/containariumd
timeout 600 env CGO_ENABLED=0 go build -p 2 -tags containarium_client -ldflags "$flags" -o "$output_root/containarium" ./cmd/containarium
timeout 600 env CGO_ENABLED=0 go build -p 2 -ldflags "$flags" -o "$output_root/containarium-mcp" ./cmd/mcp-server
cp "$profile_path" "$output_root/source-profile.json"
(cd "$output_root" && sha256sum containariumd containarium containarium-mcp source-profile.json > SHA256SUMS)
cat "$output_root/SHA256SUMS"
