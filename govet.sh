#! /bin/bash
set -e

OUTPUT=$(find . -maxdepth 1 -mindepth 1 -type d \
  | egrep -v '^\.\/vendor|^.\/docs|^\.\/\..*' \
  | xargs -I {} -P 4 go tool vet -all -composites=false -unreachable=false -tests=false -shadow {})

if [[ $OUTPUT ]]; then
  printf "govet found some issues:\n\n"
  echo "$OUTPUT"
  exit 1
fi
