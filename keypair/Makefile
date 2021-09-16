BENCHTIME?=1s

bench:
	rm -f *.prof *.test
	git checkout master
	git checkout - -- benchmarks_test.go
	go test -run=- -benchmem -bench=. -benchtime=$(BENCHTIME) -v -memprofile=mem-master.prof -cpuprofile=cpu-master.prof > out-master.prof
	git checkout -
	go test -run=- -benchmem -bench=. -benchtime=$(BENCHTIME) -v -memprofile=mem-branch.prof -cpuprofile=cpu-branch.prof > out-branch.prof
	go run golang.org/x/perf/cmd/benchstat@latest -delta-test=none out-master.prof out-branch.prof
