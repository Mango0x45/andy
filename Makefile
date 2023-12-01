.POSIX:

repl:
	@find . -name '*.go' -not -name '*_test.go' -exec rlwrap -H .andy-hist -- go run {} +
