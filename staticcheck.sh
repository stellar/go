#! /bin/bash
set -e

# Check if staticcheck is installed, if not install it.
command -v staticcheck >/dev/null 2>&1 || go get honnef.co/go/tools/cmd/staticcheck

printf "Running staticcheck...\n"

ls -d */ \
  | egrep -v '^vendor|^docs' \
  | xargs -I {} -P 4 staticcheck -tests=false -ignore="*:ST1003,SA1019" ./{}...
