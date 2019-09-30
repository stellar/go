#! /bin/bash
set -e

printf "Running go vet...\n"
go vet -all -composites=false -unreachable=false -tests=false ./...

printf "Running go vet shadow...\n"
command -v shadow >/dev/null 2>&1 || (
  dir=$(mktemp -d)
  pushd $dir
  go mod init tool
  go get golang.org/x/tools/go/analysis/passes/shadow/cmd/shadow
  popd
)

# The go vet -vettool option was added in 1.12, and broken in the initial 1.13
# release. Until it is fixed we must call vettool's directly as a work around.
# https://github.com/golang/go/issues/34053
if [[ $GOLANG_VERSION = 1.12.* ]]; then
  go vet -vettool=$(which shadow) ./...
else
  shadow ./...
fi
