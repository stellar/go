#! /bin/bash
set -e

DEVEL_GO_VERSION='1.11'

# Only run format checks on the recommended developer version of Go
if [ "$TRAVIS_GO_VERSION" != "$DEVEL_GO_VERSION" ]; then
    printf "Skipping gofmt checks for this version of Go...\n"
    exit 0
fi

printf "Running gofmt checks...\n"
OUTPUT=$(ls -d */ \
  | egrep -v '^vendor|^docs' \
  | xargs -I {} -P 4 gofmt -d {})

if [[ $OUTPUT ]]; then
  printf "gofmt found unformatted files:\n\n"
  echo "$OUTPUT"
  exit 1
fi
