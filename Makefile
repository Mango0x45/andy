PREFIX = /usr/local

all: andy
andy: $(wildcard cmd/andy/*.go)
	go build ./cmd/andy

repl:
	find . -name '*.go' -not -name '*_test.go' -exec \
		rlwrap -H .andy-hist -- go run {} +

install:
	mkdir -p ${PREFIX}/bin
	cp andy ${PREFIX}/bin
