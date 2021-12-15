#! /bin/bash
set -e

printf "Running staticcheck...\n"

staticcheck='go run honnef.co/go/tools/cmd/staticcheck@2020.1.4'

ls -d */ \
  | egrep -v '^vendor|^docs' \
  | xargs -I {} $staticcheck -tests=false -checks="all,-ST1003,-SA1019,-ST1005,-ST1000,-ST1016,-S1039,-ST1021,-ST1020,-ST1019,-SA4022" ./{}...

# Check horizon for unused exported symbols (relying on the fact that it should be self-contained)
$staticcheck -unused.whole-program -checks='U*' ./services/horizon/...
