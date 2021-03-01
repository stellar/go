#!/bin/bash
set -e

apt-get clean && apt-get update && apt-get install -y stellar-horizon=$PACKAGE_VERSION

mkdir released
cd released

wget https://github.com/stellar/go/releases/download/$TAG/$TAG-darwin-amd64.tar.gz
wget https://github.com/stellar/go/releases/download/$TAG/$TAG-linux-amd64.tar.gz
wget https://github.com/stellar/go/releases/download/$TAG/$TAG-linux-arm.tar.gz
wget https://github.com/stellar/go/releases/download/$TAG/$TAG-windows-amd64.zip

tar -xvf $TAG-darwin-amd64.tar.gz
tar -xvf $TAG-linux-amd64.tar.gz
tar -xvf $TAG-linux-arm.tar.gz
unzip $TAG-windows-amd64.zip

cd -

git pull origin --tags
git checkout $TAG
# -keep: artifact directories are not removed after packaging
CIRCLE_TAG=$TAG go run -v ./support/scripts/build_release_artifacts -keep

suffixes=(darwin-amd64 linux-amd64 linux-arm windows-amd64)

for S in "${suffixes[@]}"
do
	echo $TAG-$S
    
    if [ -f "./released/$TAG-$S.tar.gz" ]; then
        shasum -a 256 ./released/$TAG-$S.tar.gz
        shasum -a 256 ./released/$TAG-$S/horizon
    else
        # windows
        shasum -a 256 ./released/$TAG-$S.zip
        shasum -a 256 ./released/$TAG-$S/horizon.exe
    fi

    if [ -f "./dist/$TAG-$S.tar.gz" ]; then
        shasum -a 256 ./dist/$TAG-$S.tar.gz
        shasum -a 256 ./dist/$TAG-$S/horizon
    else
        # windows
        shasum -a 256 ./dist/$TAG-$S.zip
        shasum -a 256 ./dist/$TAG-$S/horizon.exe
    fi
done

echo "debian package"
shasum -a 256 $(which stellar-horizon)