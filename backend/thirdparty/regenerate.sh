#!/usr/bin/env bash
# Rebuilds the vendored protobuf fork from the module cache and re-applies the patches in
# ./patches. Run it after bumping the version in ../go.mod.
#
# Why the fork exists: see ./README.md and ../../docs/BINARY_SIZE_PLAN.md.
#
# The gocql fork is NOT here: gocql is genix-orm's dependency, so it lives in
# genix-orm/thirdparty/ with its own regenerate.sh.
#
#   ./regenerate.sh                    rebuild the fork
#   ./regenerate.sh refresh-packages   re-derive the reachable package list
set -euo pipefail

cd "$(dirname "$0")"
BACKEND_DIR="$(cd .. && pwd)"
MODULE_CACHE="$(go env GOMODCACHE)"

# Version pin. Must match the `replace` target in ../go.mod.
PROTOBUF_VERSION="v1.36.11"

log() { printf '\n\033[1m%s\033[0m\n' "$*"; }

# The protobuf fork is trimmed to the packages the backend actually reaches, which is 33 of 491.
# That list is checked in rather than derived at regeneration time, because deriving it needs
# `go list` to resolve the replace target -- which is the very tree being rebuilt. Refresh it with
# `./regenerate.sh refresh-packages` while the current fork is intact and building.
PROTOBUF_PACKAGE_LIST="patches/protobuf-packages.txt"

# refresh_protobuf_packages rewrites the checked-in list from the live dependency graph. Both the
# tagged (shipping) and untagged (development) builds are unioned, so `go build` without the deploy
# tags cannot hit a package that was trimmed away.
refresh_protobuf_packages() {
	log "protobuf: refreshing ${PROTOBUF_PACKAGE_LIST} from the current build"
	local resolved
	resolved="$(
		{
			(cd "$BACKEND_DIR" && go list -deps . 2>/dev/null)
			(cd "$BACKEND_DIR" && go list -tags 'lambda.norpc,grpcnotrace' -deps . 2>/dev/null)
		} | grep '^google.golang.org/protobuf' | sed 's|google.golang.org/protobuf/*||' | sort -u
	)"
	[ -n "$resolved" ] || { echo "go list resolved no protobuf packages -- fix the build first"; exit 1; }
	printf '%s\n' "$resolved" > "$PROTOBUF_PACKAGE_LIST"
	echo "  $(wc -l < "$PROTOBUF_PACKAGE_LIST") packages"
}

# copy_package copies one package directory: .go sources minus tests, plus any go:embed assets.
copy_package() {
	local source_dir="$1" target_dir="$2"
	mkdir -p "$target_dir"
	find "$source_dir" -maxdepth 1 -type f \
		\( -name '*.go' -o -name '*.binpb' -o -name '*.txt' \) \
		! -name '*_test.go' -exec cp {} "$target_dir/" \;
}

regenerate_protobuf() {
	local source_root="${MODULE_CACHE}/google.golang.org/protobuf@${PROTOBUF_VERSION}"
	[ -d "$source_root" ] || { echo "not in module cache: $source_root (run 'go mod download' first)"; exit 1; }

	[ -s "$PROTOBUF_PACKAGE_LIST" ] || { echo "missing $PROTOBUF_PACKAGE_LIST"; exit 1; }
	log "protobuf ${PROTOBUF_VERSION}: trimming to $(wc -l < "$PROTOBUF_PACKAGE_LIST") packages"
	rm -rf protobuf
	mkdir -p protobuf
	cp "$source_root/go.mod" "$source_root/go.sum" "$source_root/LICENSE" "$source_root/PATENTS" protobuf/

	local package_count=0
	while read -r package_path; do
		[ -n "$package_path" ] || continue
		copy_package "${source_root}/${package_path}" "protobuf/${package_path}"
		package_count=$((package_count + 1))
	done < "$PROTOBUF_PACKAGE_LIST"
	echo "  ${package_count} packages, $(find protobuf -name '*.go' | wc -l) files"

	# git apply resolves patch paths against the REPOSITORY ROOT and silently ignores anything
	# outside the current subdirectory -- run from ./protobuf it applies nothing and still exits 0.
	# So it is run from the root with --directory. The verification below exists because that
	# failure was invisible: the binary grew back by 11 MB with no error anywhere.
	log "protobuf: applying patches"
	local repository_root
	repository_root="$(git -C "$BACKEND_DIR" rev-parse --show-toplevel)"
	for patch_file in patches/protobuf-*.patch; do
		echo "  $(basename "$patch_file")"
		git -C "$repository_root" apply --directory=backend/thirdparty/protobuf "$(pwd)/$patch_file"
	done

	# Never trust that the patches landed. Every patched hunk carries this marker.
	local patched_file_count
	patched_file_count="$(grep -rl 'PATCHED (genix)' protobuf | wc -l)"
	if [ "$patched_file_count" -ne 3 ]; then
		echo "expected 3 patched files, found ${patched_file_count} -- the patches did not apply"
		exit 1
	fi
	echo "  verified ${patched_file_count} patched files"
}


case "${1:-all}" in
refresh-packages) refresh_protobuf_packages ;;
protobuf | all) regenerate_protobuf ;;
*) echo "unknown fork: $1"; exit 1 ;;
esac

log "done. Verify with:"
echo "  cd $BACKEND_DIR && go build ./... && go vet ./... && go test ./thirdparty/"
