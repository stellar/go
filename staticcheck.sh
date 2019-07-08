#! /bin/bash
set -e

# Check if staticcheck is installed, if not install it.
command -v staticcheck >/dev/null 2>&1 || go get honnef.co/go/tools/cmd/staticcheck

printf "Running staticcheck...\n"

ls -d */ \
  | egrep -v '^vendor|^docs' \
  | xargs -I {} staticcheck -tests=false -checks="all,-ST1003,-SA1019,-ST1005,-ST1000,-ST1016" ./{}...
