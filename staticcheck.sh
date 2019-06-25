#! /bin/bash
set -e

printf "Running staticcheck...\n"

ls -d */ \
  | egrep -v '^vendor|^docs' \
  | xargs -I {} -P 4 staticcheck -tests=false -ignore="*:ST1003,SA1019" ./{}...
