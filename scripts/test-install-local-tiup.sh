#!/bin/sh
set -eu
repo_root=$(CDPATH='' cd -P -- "$(dirname -- "$0")/.." && pwd)
test_root=$(mktemp -d)
trap 'rm -rf "$test_root"' EXIT
source_dir=$test_root/bundle
mkdir -p "$source_dir/profiles" "$source_dir/metrics"
cp "$repo_root/scripts/install-local-tiup.sh" "$source_dir/install.sh"
for file in tiup diag diag-redact scraper insight; do
  printf '#!/bin/sh\nprintf "test version\\n"\n' > "$source_dir/$file"
  chmod 755 "$source_dir/$file"
done
for file in info.toml LICENSE LICENSE-TIUP README.md TIUP-README.md INSTALL.md; do printf 'resource\n' > "$source_dir/$file"; done
printf 'source_commit=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n' > "$source_dir/BUILD.txt"
printf 'source_commit=bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb\n' > "$source_dir/TIUP-BUILD.txt"
printf 'profile\n' > "$source_dir/profiles/default.toml"
printf 'metrics\n' > "$source_dir/metrics/default.toml"
manifest() {
  (
    cd "$source_dir"
    find . -type f ! -name PAYLOAD-SHA256SUMS ! -name ip-map.json ! -name user-note.txt -print | sort > "$test_root/payload.list"
    while IFS= read -r file; do sha256sum "$file"; done < "$test_root/payload.list" > PAYLOAD-SHA256SUMS
  )
}
manifest
printf 'private customer data\n' > "$source_dir/ip-map.json"
printf 'private customer data\n' > "$source_dir/profiles/user-note.txt"

# Fresh install, spaces in the prefix, no collected/private data copied, rerun.
prefix="$test_root/fresh prefix"
sh "$source_dir/install.sh" --prefix "$prefix" > "$test_root/fresh.log"
target=$prefix/bin/tiup
test -L "$target"
release=$(dirname "$(readlink "$target")")
test -f "$release/info.toml"
test -f "$release/profiles/default.toml"
test ! -e "$release/ip-map.json"
test ! -e "$release/profiles/user-note.txt"
sh "$source_dir/install.sh" --prefix "$prefix" > /dev/null
test ! -e "$prefix/bin/tiup.before-diag-redact"
sh "$source_dir/install.sh" --prefix "$prefix" --uninstall > /dev/null
test ! -e "$target"
test -d "$release"

# Existing command requires explicit replacement; backup restored exactly.
prefix=$test_root/existing
mkdir -p "$prefix/bin"
printf '#!/bin/sh\nprintf "original version\\n"\n' > "$prefix/bin/tiup"
chmod 755 "$prefix/bin/tiup"
cp "$prefix/bin/tiup" "$test_root/original"
if sh "$source_dir/install.sh" --prefix "$prefix" > /dev/null 2>&1; then exit 1; fi
cmp "$prefix/bin/tiup" "$test_root/original"
PATH="$prefix/bin:$PATH" sh "$source_dir/install.sh" --replace-tiup > /dev/null
test -L "$prefix/bin/tiup"
cmp "$prefix/bin/tiup.before-diag-redact" "$test_root/original"
sh "$source_dir/install.sh" --prefix "$prefix" > /dev/null
cmp "$prefix/bin/tiup.before-diag-redact" "$test_root/original"

# An upgrade preserves the original backup and old version directory.
old_release=$(dirname "$(readlink "$prefix/bin/tiup")")
printf 'source_commit=cccccccccccccccccccccccccccccccccccccccc\n' > "$source_dir/BUILD.txt"
manifest
sh "$source_dir/install.sh" --prefix "$prefix" > /dev/null
new_release=$(dirname "$(readlink "$prefix/bin/tiup")")
test "$old_release" != "$new_release"
test -d "$old_release"
cmp "$prefix/bin/tiup.before-diag-redact" "$test_root/original"
printf 'corruption\n' >> "$new_release/info.toml"
if sh "$source_dir/install.sh" --prefix "$prefix" > /dev/null 2>&1; then exit 1; fi
test "$(dirname "$(readlink "$prefix/bin/tiup")")" = "$new_release"
sh "$source_dir/install.sh" --prefix "$prefix" --uninstall > /dev/null
cmp "$prefix/bin/tiup" "$test_root/original"

# Bad payload, directory collision, modified link and a concurrent installer fail.
printf 'corruption\n' >> "$source_dir/tiup"
if sh "$source_dir/install.sh" --prefix "$test_root/bad" > /dev/null 2>&1; then exit 1; fi
test ! -e "$test_root/bad/bin/tiup"
cp "$source_dir/diag" "$source_dir/tiup"
manifest
mkdir -p "$test_root/directory/bin/tiup"
if sh "$source_dir/install.sh" --prefix "$test_root/directory" --replace-tiup > /dev/null 2>&1; then exit 1; fi
mkdir -p "$test_root/locked/lib/diag-redact/.install-lock"
if sh "$source_dir/install.sh" --prefix "$test_root/locked" > /dev/null 2>&1; then exit 1; fi
if sh "$source_dir/install.sh" --prefix "$prefix" --uninstall > /dev/null 2>&1; then exit 1; fi
cmp "$prefix/bin/tiup" "$test_root/original"

# If publishing the new command link fails after backup, restore the old binary.
prefix=$test_root/rollback
mkdir -p "$prefix/bin" "$test_root/fail-tools"
cp "$test_root/original" "$prefix/bin/tiup"
real_mv=$(command -v mv)
cat > "$test_root/fail-tools/mv" <<'SH'
#!/bin/sh
if [ "$1" = -Tf ] && [ "$3" = "$FAIL_TARGET" ]; then
  printf 'publish failed\n' > "$FAIL_MARKER"
  exit 91
fi
exec "$REAL_MV" "$@"
SH
chmod 755 "$test_root/fail-tools/mv"
if PATH="$test_root/fail-tools:$PATH" REAL_MV="$real_mv" FAIL_TARGET="$prefix/bin/tiup" FAIL_MARKER="$test_root/publish-failed" \
  sh "$source_dir/install.sh" --prefix "$prefix" --replace-tiup > /dev/null 2>&1; then exit 1; fi
test -f "$test_root/publish-failed"
cmp "$prefix/bin/tiup" "$test_root/original"
test ! -e "$prefix/bin/tiup.before-diag-redact"
printf 'Installer regression tests PASS\n'
