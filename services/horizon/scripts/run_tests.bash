#! /usr/bin/env bash

set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
GOTOP="$( cd "$DIR/../../../../../../.." && pwd )"
PACKAGES=$(find $GOTOP/src/github.com/stellar/go/services/horizon -type d | grep -o 'github.com/.*')

for i in $PACKAGES; do
	has_tests=`ls -1 $GOTOP/src/$i/*_test.go 2>/dev/null | wc -l`

	if [ $has_tests != 0 ]; then
		go test $i
	else
		echo "skipping $i, no tests"
	fi
done

echo "All tests pass!"
