#!/bin/sh
# Build on Linux with dpkg-deb. Cross-compiles the self-contained Go binary.
set -eu
VERSION=${VERSION:-0.4.0}
ARCH=${ARCH:-amd64}
case "$ARCH" in amd64|arm64) ;; *) echo 'ARCH must be amd64 or arm64' >&2; exit 1;; esac
case "$VERSION" in ''|*[!0-9A-Za-z.+:~-]*) echo 'Invalid VERSION' >&2; exit 1;; esac
command -v dpkg-deb >/dev/null 2>&1 || { echo 'Install dpkg-dev on Debian/Ubuntu to build .deb packages.' >&2; exit 1; }
cd "$(dirname "$0")/.."
stage=$(mktemp -d)
trap 'rm -rf "$stage"' EXIT HUP INT TERM
mkdir -p "$stage/DEBIAN" "$stage/usr/bin" "$stage/usr/share/doc/pwprofiler" dist
CGO_ENABLED=0 GOOS=linux GOARCH="$ARCH" go build -trimpath -ldflags "-s -w -X main.version=$VERSION" -o "$stage/usr/bin/pwprofiler" ./cmd/pwprofiler
cp LICENSE "$stage/usr/share/doc/pwprofiler/copyright"
cp README.md "$stage/usr/share/doc/pwprofiler/README.md"
cat > "$stage/DEBIAN/control" <<CONTROL
Package: pwprofiler
Version: $VERSION
Section: utils
Priority: optional
Architecture: $ARCH
Maintainer: Password Profiler contributors <noreply@github.com>
Homepage: https://github.com/praneeth132006/Password-Profiler
Description: Local password-audit wordlist workspace
 Policy-aware text wordlists with an interactive console, file CLI and
 embedded localhost browser interface for authorized password auditing.
CONTROL
chmod 0755 "$stage" "$stage/DEBIAN" "$stage/usr/bin/pwprofiler"
dpkg-deb --root-owner-group --build "$stage" "dist/pwprofiler_${VERSION}_${ARCH}.deb"
