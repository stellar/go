#! /usr/bin/env bash
set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
GOTOP="$( cd "$DIR/../../../../../../.." && pwd )"
DIST="$GOTOP/dist"
VERSION=$(git describe --always --dirty --tags)
GOARCH=amd64
CURRENT_GOOS="$(go run $DIR/current_os.go)"

build() {
	GOOS=$1
	RELEASE="horizon-$VERSION-$GOOS-$GOARCH"
	PKG_DIR="$DIST/$RELEASE"

	# do the actual build
	GOOS=$GOOS GOARCH=$GOARCH go build -ldflags "-X main.version=$VERSION" -o "$GOTOP/bin/$(srcBin $GOOS)" github.com/stellar/go/services/horizon/cmd/horizon

	# make package directory
	rm -rf $PKG_DIR
	mkdir -p $PKG_DIR
	cp $GOTOP/bin/$(srcBin $GOOS) $PKG_DIR/$(destBin $GOOS)
	cp $DIR/../LICENSE.txt $PKG_DIR/
	cp $DIR/../README.md $PKG_DIR/

	# TODO: add platform specific install intstructions

	# zip/tar package directory
	pkg $GOOS $RELEASE
}

srcBin() {
	GOOS=$1
	BIN="horizon-$GOOS-$GOARCH"

	if [ "$GOOS" = "windows" ]; then
		BIN+=".exe"
	fi

	echo $BIN
}

destBin() {
	if [ "$1" = "windows" ]; then
		echo "horizon.exe"
	else
		echo "horizon"
	fi
}

pkg() {
	GOOS=$1
	RELEASE=$2

	if [ "$GOOS" = "windows" ]; then
		pushd $DIST
		zip $RELEASE.zip $RELEASE/*
		popd
	else
		tar -czf $DIST/$RELEASE.tar.gz -C $DIST $RELEASE
	fi

	rm -rf $DIST/$RELEASE
}

build darwin
build linux
build windows
