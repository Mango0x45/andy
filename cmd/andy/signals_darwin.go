package main

import (
	"maps"
	"os"
	"syscall"
)

func init() {
	maps.Copy(signals, map[string]os.Signal{
		"sigio":     syscall.SIGIO,
		"sigiot":    syscall.SIGIOT,
		"sigprof":   syscall.SIGPROF,
		"sigsys":    syscall.SIGSYS,
		"sigvtalrm": syscall.SIGVTALRM,
		"sigwinch":  syscall.SIGWINCH,
	})
}
