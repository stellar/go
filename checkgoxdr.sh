#! /bin/bash
set -e

printf "Running make gxdr/xdr_generated.go...\n"
make gxdr/xdr_generated.go

printf "Checking for no diff...\n"
git diff --exit-code || (echo "Files changed after running make gxdr/xdr_generated.go. Run make gxdr/xdr_generated.go locally and update generated files." && exit 1)
