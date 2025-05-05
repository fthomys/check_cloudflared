#!/bin/bash

set -euo pipefail

REPO="fthomys/check_cloudflared"
DIST_DIR="/var/www/packages.fthomys.me/debian"
ARCH="amd64"

: "${GPG_KEY_ID:?Please export GPG_KEY_ID first (e.g. export GPG_KEY_ID=ABCD1234)}"

VERSION=$(curl -s https://api.github.com/repos/$REPO/releases/latest | jq -r .tag_name | sed 's/^v//')
if [[ -z "$VERSION" ]]; then
    echo "Error: Could not determine version."
    exit 1
fi

WORKDIR=$(mktemp -d)
trap 'rm -rf "$WORKDIR"' EXIT

git clone --depth 1 --branch "v$VERSION" https://github.com/$REPO.git "$WORKDIR/src"
cd "$WORKDIR/src"
go build -o check_cloudflared

PKG_NAME="monitoring-check-cloudflared"
PKG_ROOT="${WORKDIR}/${PKG_NAME}_${VERSION}_$ARCH"
mkdir -p "$PKG_ROOT/DEBIAN"
mkdir -p "$PKG_ROOT/usr/lib/nagios/plugins"

cat > "$PKG_ROOT/DEBIAN/control" <<EOF
Package: $PKG_NAME
Version: $VERSION
Section: monitoring
Priority: optional
Architecture: $ARCH
Maintainer: Fabian Thomys <git@fthomys.me>
Description: Icinga plugin to check cloudflared installation and updates
Depends: curl
EOF

cp check_cloudflared "$PKG_ROOT/usr/lib/nagios/plugins/"
chmod 755 "$PKG_ROOT/usr/lib/nagios/plugins/check_cloudflared"

DEB_NAME="${PKG_NAME}_${VERSION}_${ARCH}.deb"
fakeroot dpkg-deb --build "$PKG_ROOT" "$WORKDIR/$DEB_NAME"

mkdir -p "$DIST_DIR/pool/"
mv "$WORKDIR/$DEB_NAME" "$DIST_DIR/pool/"

cd "$DIST_DIR"
mkdir -p dists/stable/main/binary-amd64

dpkg-scanpackages pool /dev/null > dists/stable/main/binary-amd64/Packages
gzip -c9 dists/stable/main/binary-amd64/Packages > dists/stable/main/binary-amd64/Packages.gz

cat > dists/stable/Release <<EOF
Origin: fthomys
Label: fthomys
Suite: stable
Codename: stable
Architectures: amd64
Components: main
EOF

gpg --batch --yes -abs -o dists/stable/Release.gpg --local-user "$GPG_KEY_ID" dists/stable/Release
gpg --batch --yes --clearsign -o dists/stable/InRelease --local-user "$GPG_KEY_ID" dists/stable/Release

gpg --armor --export "$GPG_KEY_ID" > "$DIST_DIR/fthomys.gpg.key"

echo "[✓] Version $VERSION built and published successfully!"
