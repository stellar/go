#! /bin/bash
set -e

# Only run format checks on the recommended developer version of Go
if [ $TRAVIS_GO_VERSION != '1.11' ]; then
    exit 0
fi

OUTPUT=$(ls -d */ \
  | egrep -v '^vendor|^docs' \
  | xargs -I {} -P 4 gofmt -l {})

if [[ $OUTPUT ]]; then
  printf "gofmt found unformatted files:\n\n"
  echo "$OUTPUT"
  exit 1
fi
