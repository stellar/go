#! /bin/bash
set -e

go mod tidy
git diff --exit-code -- go.mod || (echo "Go file go.mod is dirty, update the file with 'go mod tidy' locally." && exit 1)
if [[ ! $GOLANG_VERSION = 1.11.* ]]; then
  git diff --exit-code -- go.sum || (echo "Go file go.sum is dirty, update the file with 'go mod tidy' locally." && exit 1)
else
  echo "Skipping go.sum check for Go 1.11.* because it doesn't generate go.sum files consistently with other versions of Go."
fi
diff -u go.list <(go list -m all) || (echo "Go dependencies have changed, update the go.list file with 'go list -m all > go.list' locally." && exit 1)
go mod verify || (echo "One or more Go dependencies failed verification. Either a version is no longer available, or the author or someone else has modified the version so it no longer points to the same code." && exit 1)
