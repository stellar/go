#! /bin/bash
set -e

# Check if staticcheck is installed, if not install it.
command -v staticcheck >/dev/null 2>&1 || (
  dir=$(mktemp -d)
  pushd $dir
  go mod init tool
  go get honnef.co/go/tools/cmd/staticcheck@2019.2
  popd
)

printf "Running staticcheck...\n"

ls -d */ \
  | egrep -v '^vendor|^docs' \
  | xargs -I {} staticcheck -tests=false -checks="all,-ST1003,-SA1019,-ST1005,-ST1000,-ST1016" ./{}...
