#! /bin/bash
set -e

go mod tidy
git diff --exit-code || (echo "Go mod file is dirty, update the go.mod file with 'go mod tidy' locally." && exit 1)
diff -u go.list <(go list -m all) || (echo "Go dependencies have changed, update the go.list file with 'go list -m all > go.list' locally." && exit 1)
go mod verify || (echo "One or more Go dependencies failed verification. Either a version is no longer available, or the author or someone else has modified the version so it no longer points to the same code." && exit 1)
