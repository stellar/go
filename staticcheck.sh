#! /bin/bash
set -e

gover=$(go version | { read _ _ gover _; printf $gover; })

version='2020.1.4'
if [[ "$gover" = "go1.18"* ]]; then
  version='d5c28addcbbbafca0b9a0f9ad8957912e9371015'
fi

staticcheck='go run honnef.co/go/tools/cmd/staticcheck@'"$version"

printf "Running staticcheck $version...\n"

ls -d */ \
  | egrep -v '^vendor|^docs' \
  | xargs -I {} $staticcheck -tests=false -checks="all,-ST1003,-SA1019,-ST1005,-ST1000,-ST1016,-S1039,-ST1021,-ST1020,-ST1019,-SA4022" ./{}...

# Whole program unused checks were removed from staticcheck in newer versions,
# so this check is being sunset and will be removed once Go 1.18 is released and
# a proper release of staticcheck is released that supports it.
if [ "$version" = "2020.1.4" ]; then
  # Check horizon for unused exported symbols (relying on the fact that it should be self-contained)
  $staticcheck -unused.whole-program -checks='U*' ./services/horizon/...
fi
