#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tiup_root="$(cd "${1:?Usage: build-local-tiup-redact.sh <tiup-source> [artifact-directory]}" && pwd)"
artifact_root="${2:-$repo_root/package/diag-redact-local}"
mkdir -p "$artifact_root"
artifact_root="$(cd "$artifact_root" && pwd)"
diag_sha="$(git -C "$repo_root" rev-parse HEAD)"
tiup_sha="$(git -C "$tiup_root" rev-parse HEAD)"
tiup_branch="$(git -C "$tiup_root" branch --show-current)"
if ! git -C "$tiup_root" diff --quiet HEAD --; then
  echo "TiUP tracked files differ from HEAD; commit verified source before packaging." >&2
  exit 1
fi
for target in linux-amd64 linux-arm64; do
  bundle="diag-redact-local-${diag_sha:0:12}-${tiup_sha:0:12}-${target}"
  for item in "$bundle" "$bundle.tar.gz" "$bundle.zip" "$bundle.sha256"; do
    if test -e "$artifact_root/$item"; then
      echo "Artifact already exists: $item" >&2
      exit 1
    fi
  done
done

build_tmp="$(mktemp -d)"
trap 'rm -rf "$build_tmp"' EXIT
bash "$repo_root/scripts/build-cn2-redacted.sh" "$build_tmp"
for target in linux-amd64 linux-arm64; do
bundle="diag-redact-local-${diag_sha:0:12}-${tiup_sha:0:12}-${target}"
mv "$build_tmp/diag-cn2-ip-redaction-${diag_sha:0:12}-${target}" "$build_tmp/$bundle"
(
  cd "$tiup_root"
  CGO_ENABLED=0 GOFLAGS='' GOOS=linux GOARCH="${target#linux-}" go build -trimpath \
    -ldflags "-s -w -X github.com/pingcap/tiup/pkg/version.GitHash=$tiup_sha -X github.com/pingcap/tiup/pkg/version.GitRef=$tiup_branch" \
    -o "$build_tmp/$bundle/tiup" .
)
cp "$tiup_root/LICENSE" "$build_tmp/$bundle/LICENSE-TIUP"
cp "$tiup_root/doc/user/diag-redact.md" "$build_tmp/$bundle/TIUP-README.md"
cp "$repo_root/scripts/install-local-tiup.sh" "$build_tmp/$bundle/install.sh"
cp "$repo_root/docs/manual/INSTALL.md" "$build_tmp/$bundle/INSTALL.md"
chmod 755 "$build_tmp/$bundle/install.sh"
{
  echo "source_commit=$tiup_sha"
  echo "source_branch=$tiup_branch"
  echo "target=$target"
  go version
} > "$build_tmp/$bundle/TIUP-BUILD.txt"
(
  cd "$build_tmp/$bundle"
  find . -type f ! -name PAYLOAD-SHA256SUMS -print | LC_ALL=C sort > "$build_tmp/payload.list"
  while IFS= read -r file; do
    shasum -a 256 "$file"
  done < "$build_tmp/payload.list" > PAYLOAD-SHA256SUMS
)
COPYFILE_DISABLE=1 tar -C "$build_tmp" -czf "$artifact_root/$bundle.tar.gz" "$bundle"
mv "$build_tmp/$bundle" "$artifact_root/$bundle"
(
  cd "$artifact_root"
  shasum -a 256 "$bundle.tar.gz" > "$bundle.sha256"
  zip -q "$bundle.zip" "$bundle.tar.gz" "$bundle.sha256"
)
echo "Bundle: $artifact_root/$bundle.zip"
done
