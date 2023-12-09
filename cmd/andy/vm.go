package main

import (
	"io"
	"os"
	"os/signal"
	"strconv"
	"syscall"
)

type context struct {
	in       io.Reader
	out, err io.Writer
	scope    map[string][]string
}

type vm struct {
	file        bool
	interactive bool
	sigsHandled bool
}

type function struct {
	args []string
	body astProgram
}

var (
	globalFuncMap     map[string]function
	globalVariableMap map[string][]string
)

func init() {
	globalFuncMap = make(map[string]function, 64)
	globalVariableMap = map[string][]string{
		"_":      {}, // Other shells export this
		"pid":    {strconv.Itoa(os.Getpid())},
		"ppid":   {strconv.Itoa(os.Getppid())},
		"status": {"0"},
	}
}

func (vm *vm) run(prog astProgram) {
	if vm.file {
		globalVariableMap["args"] = os.Args[1:]
	}

	if !vm.sigsHandled {
		ch := make(chan os.Signal, 1)
		signal.Notify(
			ch,
			syscall.SIGABRT,
			syscall.SIGALRM,
			syscall.SIGBUS,
			syscall.SIGCHLD,
			syscall.SIGCONT,
			syscall.SIGFPE,
			syscall.SIGHUP,
			syscall.SIGILL,
			syscall.SIGINT,
			syscall.SIGIO,
			syscall.SIGKILL,
			syscall.SIGPIPE,
			syscall.SIGPROF,
			syscall.SIGPWR,
			syscall.SIGQUIT,
			syscall.SIGSEGV,
			syscall.SIGSTKFLT,
			syscall.SIGSTOP,
			syscall.SIGSYS,
			syscall.SIGTERM,
			syscall.SIGTRAP,
			syscall.SIGTSTP,
			syscall.SIGTTIN,
			syscall.SIGTTOU,
			syscall.SIGURG,
			syscall.SIGUSR1,
			syscall.SIGUSR2,
			syscall.SIGVTALRM,
			syscall.SIGWINCH,
			syscall.SIGXCPU,
			syscall.SIGXFSZ,
		)
		go signalHandler(ch)
		vm.sigsHandled = true
	}

	var failed bool
	for _, tl := range prog {
		res := execTopLevel(tl, context{
			os.Stdin,
			os.Stdout,
			os.Stderr,
			nil,
		})
		code := int(res.ExitCode())
		globalVariableMap["status"] = []string{strconv.Itoa(code)}
		if cmdFailed(res) {
			if _, ok := res.(errExitCode); !ok {
				warn(res)
			}
			if !vm.interactive {
				failed = true
				break
			}
		}
	}
	if f, ok := globalFuncMap["sigexit"]; ok {
		res := execTopLevels(f.body, context{
			os.Stdin,
			os.Stdout,
			os.Stderr,
			map[string][]string{"_": {}},
		})
		if cmdFailed(res) {
			failed = true
		}
	}

	if failed {
		os.Exit(1)
	}
}

func signalHandler(ch <-chan os.Signal) {
	for sig := range ch {
		var s string
		switch sig {
		case syscall.SIGABRT:
			s = "sigabrt"
		case syscall.SIGALRM:
			s = "sigalrm"
		case syscall.SIGBUS:
			s = "sigbus"
		case syscall.SIGCHLD:
			s = "sigchld"
		case syscall.SIGCONT:
			s = "sigcont"
		case syscall.SIGFPE:
			s = "sigfpe"
		case syscall.SIGHUP:
			s = "sighup"
		case syscall.SIGILL:
			s = "sigill"
		case syscall.SIGINT:
			s = "sigint"
		case syscall.SIGIO:
			s = "sigio"
		case syscall.SIGKILL:
			s = "sigkill"
		case syscall.SIGPIPE:
			s = "sigpipe"
		case syscall.SIGPROF:
			s = "sigprof"
		case syscall.SIGPWR:
			s = "sigpwr"
		case syscall.SIGQUIT:
			s = "sigquit"
		case syscall.SIGSEGV:
			s = "sigsegv"
		case syscall.SIGSTKFLT:
			s = "sigstkflt"
		case syscall.SIGSTOP:
			s = "sigstop"
		case syscall.SIGSYS:
			s = "sigsys"
		case syscall.SIGTERM:
			s = "sigterm"
		case syscall.SIGTRAP:
			s = "sigtrap"
		case syscall.SIGTSTP:
			s = "sigtstp"
		case syscall.SIGTTIN:
			s = "sigttin"
		case syscall.SIGTTOU:
			s = "sigttou"
		case syscall.SIGURG:
			s = "sigurg"
		case syscall.SIGUSR1:
			s = "sigusr1"
		case syscall.SIGUSR2:
			s = "sigusr2"
		case syscall.SIGVTALRM:
			s = "sigvtalrm"
		case syscall.SIGWINCH:
			s = "sigwinch"
		case syscall.SIGXCPU:
			s = "sigxcpu"
		case syscall.SIGXFSZ:
			s = "sigxfsz"
		}

		f, ok := globalFuncMap[s]
		if !ok {
			switch sig {
			case syscall.SIGABRT:
				s = "sigiot"
			case syscall.SIGCHLD:
				s = "sigcld"
			case syscall.SIGIO:
				s = "sigpoll"
			case syscall.SIGSYS:
				s = "sigunused"
			}
			f, ok = globalFuncMap[s]
		}
		if ok {
			execTopLevels(f.body, context{
				os.Stdin,
				os.Stdout,
				os.Stderr,
				nil,
			})
		}
	}
}
