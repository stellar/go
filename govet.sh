#! /bin/bash
set -e

printf "Running go vet...\n"
go vet -all -composites=false -unreachable=false -tests=false ./...

# -vettool was added in 1.12, and broken in the initial 1.13 release
# https://github.com/golang/go/issues/34053
if [[ $GOLANG_VERSION = 1.12.* ]]; then
  printf "Running go vet shadow...\n"
  command -v shadow >/dev/null 2>&1 || (
    dir=$(mktemp -d)
    pushd $dir
    go mod init tool
    go get golang.org/x/tools/go/analysis/passes/shadow/cmd/shadow
    popd
  )

  go vet -vettool=$(which shadow) ./...
else
  echo "Skipping go vet shadow checks for this version of Go..."
fi
