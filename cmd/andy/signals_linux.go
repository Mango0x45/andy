package main

import (
	"maps"
	"os"
	"syscall"
)

func init() {
	maps.Copy(signals, map[string]os.Signal{
		"sigcld":    syscall.SIGCLD,
		"sigio":     syscall.SIGIO,
		"sigiot":    syscall.SIGIOT,
		"sigpoll":   syscall.SIGPOLL,
		"sigprof":   syscall.SIGPROF,
		"sigpwr":    syscall.SIGPWR,
		"sigstkflt": syscall.SIGSTKFLT,
		"sigsys":    syscall.SIGSYS,
		"sigunused": syscall.SIGUNUSED,
		"sigvtalrm": syscall.SIGVTALRM,
		"sigwinch":  syscall.SIGWINCH,
	})
}
