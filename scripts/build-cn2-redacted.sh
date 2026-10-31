#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"
artifact_root="${1:-$repo_root/package/cn2-ip-redaction}"
mkdir -p "$artifact_root"
artifact_root="$(cd "$artifact_root" && pwd)"
source_sha="$(git rev-parse HEAD)"
source_branch="$(git branch --show-current)"
source_version="cn2-ip-redaction-${source_sha:0:12}"
if ! git diff --quiet HEAD --; then
  echo "Tracked files differ from HEAD; commit verified source before packaging." >&2
  exit 1
fi

export CGO_ENABLED=0
export GOFLAGS=
ldflags="-s -w -X github.com/pingcap/diag/version.ReleaseVersion=$source_version -X github.com/pingcap/diag/version.GitHash=$source_sha -X github.com/pingcap/diag/version.GitBranch=$source_branch"

for target in linux-amd64 linux-arm64; do
  export GOOS="${target%-*}"
  export GOARCH="${target#*-}"
  bundle="diag-${source_version}-${target}"
  bundle_dir="$artifact_root/$bundle"
  if test -e "$bundle_dir" || test -e "$artifact_root/$bundle.tar.gz"; then
    echo "Artifact already exists: $bundle" >&2
    exit 1
  fi
  mkdir "$bundle_dir"
  for component in diag diag-redact scraper insight; do
    go build -trimpath -ldflags "$ldflags" -o "$bundle_dir/$component" "./cmd/$component"
  done
  cp configs/info.toml "$bundle_dir/"
  cp -R configs/metrics configs/profiles "$bundle_dir/"
  cp LICENSE "$bundle_dir/"
  cp docs/manual/CN2-IP-redaction.md "$bundle_dir/README.md"
  {
    echo "source_commit=$source_sha"
    echo "source_branch=$source_branch"
    echo "target=$target"
    go version
  } > "$bundle_dir/BUILD.txt"
  COPYFILE_DISABLE=1 tar -C "$artifact_root" -czf "$artifact_root/$bundle.tar.gz" "$bundle"
done
(
  cd "$artifact_root"
  shasum -a 256 "diag-${source_version}-linux-amd64.tar.gz" "diag-${source_version}-linux-arm64.tar.gz" > SHA256SUMS
)
echo "Artifacts: $artifact_root"
