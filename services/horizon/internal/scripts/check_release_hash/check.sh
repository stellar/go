#!/bin/bash
# apt-get update && apt-get install -y stellar-horizon

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
CIRCLE_TAG=$TAG go run -v ./support/scripts/build_release_artifacts

suffixes=(darwin-amd64.tar.gz linux-amd64.tar.gz linux-arm.tar.gz windows-amd64.zip)

for S in "${suffixes[@]}"
do
	echo $TAG-$S
    shasum -a 256 ./released/$TAG-$S
    shasum -a 256 ./dist/$TAG-$S
done