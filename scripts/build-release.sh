#!/bin/sh
# Builds the release archives and their checksums into dist/.
#
# Only the Go toolchain is used. Nothing is published: the files are left
# in dist/ for a person to inspect and upload.
#
# The tasks are carried inside the executable, so an archive holds one
# file that works on its own. A task cut from another repository still
# needs that repository, which it says when it cannot find one.
#
# Usage: scripts/build-release.sh [version]
set -eu

cd "$(dirname "$0")/.." || exit 2

version=${1:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}
out=dist
rm -rf "$out"
mkdir -p "$out"

export CGO_ENABLED=0

targets="darwin/arm64 darwin/amd64 linux/arm64 linux/amd64 windows/amd64"

for target in $targets; do
	os=${target%/*}
	arch=${target#*/}
	name="specgap_${version}_${os}_${arch}"
	dir="$out/$name"
	mkdir -p "$dir"

	bin="specgap"
	if [ "$os" = "windows" ]; then
		bin="specgap.exe"
	fi

	GOOS="$os" GOARCH="$arch" go build \
		-trimpath \
		-ldflags "-s -w" \
		-o "$dir/$bin" ./cmd/specgap

	cp README.md LICENSE "$dir/"

	if [ "$os" = "windows" ]; then
		(cd "$out" && zip -q -r "$name.zip" "$name")
	else
		tar -czf "$out/$name.tar.gz" -C "$out" "$name"
	fi
	rm -rf "$dir"
	printf '%-46s %s\n' "$name" "$(du -h "$out"/"$name".* | cut -f1)"
done

(
	cd "$out"
	: >SHA256SUMS
	for archive in *.tar.gz *.zip; do
		if command -v sha256sum >/dev/null 2>&1; then
			sha256sum "$archive" >>SHA256SUMS
		else
			shasum -a 256 "$archive" >>SHA256SUMS
		fi
	done
)

echo
echo "dist/ holds the archives and SHA256SUMS. Nothing was published."
