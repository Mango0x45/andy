package main

import (
	"os"
	"syscall"
)

var signals = map[string]os.Signal{
	"sigabrt":   syscall.SIGABRT,
	"sigalrm":   syscall.SIGALRM,
	"sigbus":    syscall.SIGBUS,
	"sigchld":   syscall.SIGCHLD,
	"sigcont":   syscall.SIGCONT,
	"sigfpe":    syscall.SIGFPE,
	"sighup":    syscall.SIGHUP,
	"sigill":    syscall.SIGILL,
	"sigint":    syscall.SIGINT,
	"sigio":     syscall.SIGIO,
	"sigiot":    syscall.SIGIOT,
	"sigkill":   syscall.SIGKILL,
	"sigpipe":   syscall.SIGPIPE,
	"sigprof":   syscall.SIGPROF,
	"sigquit":   syscall.SIGQUIT,
	"sigsegv":   syscall.SIGSEGV,
	"sigstop":   syscall.SIGSTOP,
	"sigsys":    syscall.SIGSYS,
	"sigterm":   syscall.SIGTERM,
	"sigtrap":   syscall.SIGTRAP,
	"sigtstp":   syscall.SIGTSTP,
	"sigttin":   syscall.SIGTTIN,
	"sigttou":   syscall.SIGTTOU,
	"sigurg":    syscall.SIGURG,
	"sigusr1":   syscall.SIGUSR1,
	"sigusr2":   syscall.SIGUSR2,
	"sigvtalrm": syscall.SIGVTALRM,
	"sigwinch":  syscall.SIGWINCH,
	"sigxcpu":   syscall.SIGXCPU,
	"sigxfsz":   syscall.SIGXFSZ,
}
