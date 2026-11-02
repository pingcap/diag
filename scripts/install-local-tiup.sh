#!/bin/sh
set -eu

fail() { printf 'Error: %s\n' "$*" >&2; exit 1; }
source_dir=$(CDPATH='' cd -P -- "$(dirname -- "$0")" && pwd)
prefix=
replace=false
uninstall=false
while [ "$#" -gt 0 ]; do
  case "$1" in
    --prefix) [ "$#" -ge 2 ] || fail '--prefix requires a directory'; prefix=$2; shift 2 ;;
    --replace-tiup) replace=true; shift ;;
    --uninstall) uninstall=true; shift ;;
    -h|--help)
      printf '%s\n' 'Usage: sh install.sh [--prefix DIR] [--replace-tiup] [--uninstall]' \
        'Default: existing TiUP bin prefix, otherwise /usr/local. Existing TiUP is never replaced without --replace-tiup.'
      exit 0 ;;
    *) fail "unknown argument: $1" ;;
  esac
done
[ "$(uname -s)" = Linux ] || fail 'this installer is for the Linux bundle'
if [ -z "$prefix" ]; then
  existing=$(command -v tiup 2>/dev/null || true)
  case "$existing" in
    /usr/bin/tiup|/bin/tiup) prefix=/usr/local ;;
    /*/bin/tiup) prefix=${existing%/bin/tiup} ;;
    *) prefix=/usr/local ;;
  esac
fi
case "$prefix" in /*) ;; *) fail '--prefix must be absolute' ;; esac
prefix=${prefix%/}
[ -n "$prefix" ] || fail 'refusing to install directly under /'
mkdir -p "$prefix" || fail "cannot create $prefix; use an account with write permission"
prefix=$(CDPATH='' cd -P -- "$prefix" && pwd)
bin_dir=$prefix/bin
lib_dir=$prefix/lib/diag-redact
target=$bin_dir/tiup
backup=$bin_dir/tiup.before-diag-redact

is_managed() {
  [ -L "$target" ] || return 1
  link=$(readlink "$target")
  case "$link" in "$lib_dir/"*/tiup) ;; *) return 1 ;; esac
  release_id=${link#"$lib_dir/"}
  release_id=${release_id%/tiup}
  printf '%s\n' "$release_id" | grep -Eq '^[0-9a-f]{12}-[0-9a-f]{12}$'
}

mkdir -p "$lib_dir"
lock=$lib_dir/.install-lock
mkdir "$lock" 2>/dev/null || fail "another installation may be running: $lock"
stage=
link_tmp=
backup_moved=false
link_installed=false
cleanup() {
  if [ "$backup_moved" = true ] && [ "$link_installed" = false ]; then
    if [ ! -e "$target" ] && [ ! -L "$target" ]; then
      mv -T "$backup" "$target" || printf 'Restore backup manually: %s\n' "$backup" >&2
    fi
  fi
  [ -z "$link_tmp" ] || rm -f "$link_tmp"
  [ -z "$stage" ] || rm -rf "$stage"
  rmdir "$lock"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

if [ "$uninstall" = true ]; then
  is_managed || fail "refusing to uninstall: $target is not this installer's link"
  if [ -e "$backup" ] || [ -L "$backup" ]; then
    mv -Tf "$backup" "$target"
    printf 'Restored previous TiUP: %s\n' "$target"
  else
    rm "$target"
    printf 'Removed command link: %s\n' "$target"
  fi
  printf 'Version directories and user configuration were preserved: %s\n' "$lib_dir"
  exit 0
fi

if [ -e "$target" ] || [ -L "$target" ]; then
  [ ! -d "$target" ] || fail "command path is a directory: $target"
  if ! is_managed; then
    [ "$replace" = true ] || fail "existing TiUP at $target; rerun with --replace-tiup to back it up first"
    [ ! -e "$backup" ] && [ ! -L "$backup" ] || fail "backup already exists; refusing to overwrite: $backup"
  fi
fi

files='tiup diag diag-redact scraper insight info.toml BUILD.txt TIUP-BUILD.txt LICENSE LICENSE-TIUP README.md TIUP-README.md INSTALL.md install.sh'
for file in $files; do
  [ -f "$source_dir/$file" ] && [ ! -L "$source_dir/$file" ] || fail "missing or linked bundle file: $file"
done
for dir in profiles metrics; do
  [ -d "$source_dir/$dir" ] && [ ! -L "$source_dir/$dir" ] || fail "missing or linked resource directory: $dir"
  [ -z "$(find "$source_dir/$dir" -type l -print)" ] || fail "resource directory contains symlinks: $dir"
done
[ -f "$source_dir/PAYLOAD-SHA256SUMS" ] || fail 'bundle payload checksum manifest is missing'
(cd "$source_dir" && sha256sum -c PAYLOAD-SHA256SUMS > /dev/null) || fail 'bundle checksum verification failed'
diag_sha=$(sed -n 's/^source_commit=//p' "$source_dir/BUILD.txt")
tiup_sha=$(sed -n 's/^source_commit=//p' "$source_dir/TIUP-BUILD.txt")
printf '%s\n' "$diag_sha" | grep -Eq '^[0-9a-f]{40}$' || fail 'invalid Diag build record'
printf '%s\n' "$tiup_sha" | grep -Eq '^[0-9a-f]{40}$' || fail 'invalid TiUP build record'
release=$lib_dir/$(printf '%.12s-%.12s' "$diag_sha" "$tiup_sha")

if [ -e "$release" ] || [ -L "$release" ]; then
  [ -d "$release" ] && [ ! -L "$release" ] || fail "invalid existing version path: $release"
  cmp "$source_dir/PAYLOAD-SHA256SUMS" "$release/PAYLOAD-SHA256SUMS" || fail 'existing version has a different payload'
  (cd "$release" && sha256sum -c PAYLOAD-SHA256SUMS > /dev/null) || fail 'existing version was modified; refusing to overwrite'
else
  stage=$(mktemp -d "$lib_dir/.stage.XXXXXX")
  # Copy only release resources, never collected data, logs or private maps.
  for file in $files PAYLOAD-SHA256SUMS; do cp "$source_dir/$file" "$stage/$file"; done
  mkdir "$stage/profiles" "$stage/metrics"
  while IFS= read -r line; do
    relative=${line#*  }
    case "$relative" in
      ./profiles/*|./metrics/*)
        case "$relative" in */../*|*/./*) fail 'unsafe resource path in checksum manifest' ;; esac
        mkdir -p "$stage/$(dirname "$relative")"
        cp "$source_dir/$relative" "$stage/$relative" ;;
    esac
  done < "$source_dir/PAYLOAD-SHA256SUMS"
  find "$stage" -type d -exec chmod 755 {} +
  find "$stage" -type f -exec chmod 644 {} +
  for file in tiup diag diag-redact scraper insight install.sh; do chmod 755 "$stage/$file"; done
  (cd "$stage" && sha256sum -c PAYLOAD-SHA256SUMS > /dev/null) || fail 'installed payload verification failed'
  mv -T "$stage" "$release"
  stage=
fi

"$release/tiup" --version
"$release/tiup" diag-redact --version
mkdir -p "$bin_dir"
link_tmp=$(mktemp "$bin_dir/.tiup-link.XXXXXX")
rm "$link_tmp"
ln -s "$release/tiup" "$link_tmp"
if { [ -e "$target" ] || [ -L "$target" ]; } && ! is_managed; then
  backup_moved=true
  mv -T "$target" "$backup"
fi
mv -Tf "$link_tmp" "$target"
link_tmp=
link_installed=true
printf 'Installed: %s\nResources: %s\n' "$target" "$release"
if [ -e "$backup" ] || [ -L "$backup" ]; then printf 'Previous TiUP backup: %s\n' "$backup"; fi
printf 'Run: %s diag-redact collect <cluster> ...\n' "$target"
case ":$PATH:" in
  *":$bin_dir:"*) printf 'The command directory is already in PATH; no export is needed.\n' ;;
  *) printf 'This prefix is outside the current PATH; use the absolute command path above.\n' ;;
esac
printf 'Uninstall and restore: sh %s/install.sh --prefix %s --uninstall\n' "$release" "$prefix"
