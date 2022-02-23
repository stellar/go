#! /bin/bash
set -e

version='d5c28addcbbbafca0b9a0f9ad8957912e9371015'
staticcheck='go run honnef.co/go/tools/cmd/staticcheck@'"$version"

printf "Running staticcheck $version...\n"

ls -d */ \
  | egrep -v '^vendor|^docs' \
  | xargs -I {} $staticcheck -tests=false -checks="all,-ST1003,-SA1019,-ST1005,-ST1000,-ST1016,-S1039,-ST1021,-ST1020,-ST1019,-SA4022" ./{}...
