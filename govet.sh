#! /bin/bash
set -e

# TODO: Build and install shadow analyzer for go vet in 1.12+
# See https://github.com/golang/go/issues/29260
# For now we just skip the shadow checking for Go 1.12+
if [ "$TRAVIS_GO_VERSION" = '1.9' ] || [ "$TRAVIS_GO_VERSION" = '1.10' ] || [ "$TRAVIS_GO_VERSION" = '1.11' ]; then
GOVET_SHADOW='-shadow'
GOVET_TOOL='tool'
fi

OUTPUT=$(ls -d */ \
  | egrep -v '^vendor|^docs' \
  | xargs -I {} -P 4 go $GOVET_TOOL vet -all -composites=false -unreachable=false -tests=false $GOVET_SHADOW {})

if [[ $OUTPUT ]]; then
  printf "govet found some issues:\n\n"
  echo "$OUTPUT"
  exit 1
fi
