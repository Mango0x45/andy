package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"os/user"
	"slices"
	"strconv"
	"strings"
	"syscall"

	"git.sr.ht/~mango/andy/pkg/stack"
	"git.sr.ht/~mango/opts"
)

type builtin func(cmd *exec.Cmd, ctx context) uint8

var (
	builtins      map[string]builtin
	dirStack      = stack.New[string](64)
	reservedNames = []string{"cdstack", "pid", "ppid", "status"}
)

func init() {
	builtins = map[string]builtin{
		"call":  cmdCall,
		"cd":    cmdCd,
		"echo":  cmdEcho,
		"eval":  cmdEval,
		"exec":  cmdExec,
		"exit":  cmdExit,
		"false": cmdFalse,
		"quote": cmdQuote,
		"read":  cmdRead,
		"set":   cmdSet,
		"true":  cmdTrue,
		"type":  cmdType,
		"umask": cmdUmask,
	}
}

func cmdCall(cmd *exec.Cmd, ctx context) uint8 {
	var bflag, cflag bool
	usage := func() uint8 {
		fmt.Fprintln(cmd.Stderr, "Usage: call [-bc] command [argument ...]")
		return 1
	}

	flags, optind, err := opts.GetLong(cmd.Args, []opts.LongOpt{
		{Short: 'b', Long: "builtin", Arg: opts.None},
		{Short: 'c', Long: "command", Arg: opts.None},
	})
	if err != nil {
		cmdErrorf(cmd, "%s", err)
		return usage()
	}

	for _, f := range flags {
		switch f.Key {
		case 'b':
			bflag = true
		case 'c':
			cflag = true
		}
	}

	rest := cmd.Args[optind:]
	if len(rest) == 0 {
		return usage()
	}
	f, xs := rest[0], rest[1:]

	if !bflag && !cflag {
		bflag = true
		cflag = true
	}

	if bflag {
		if b, ok := builtins[f]; ok {
			cmd.Args = rest
			return b(cmd, ctx)
		}
	}

	if cflag {
		c := exec.Command(f, xs...)
		c.Stdin, c.Stdout, c.Stderr = cmd.Stdin, cmd.Stdout, cmd.Stderr
		c.ExtraFiles = cmd.ExtraFiles
		err = c.Run()
		code := c.ProcessState.ExitCode()

		if err != nil && code == -1 {
			return cmdErrorf(cmd, "%s", err)
		}
		return uint8(code)
	}

	return 1
}

func cmdCd(cmd *exec.Cmd, _ context) uint8 {
	cmd.Args = shiftDashDash(cmd.Args)
	defer func() {
		globalVariableMap["cdstack"] = dirStack
	}()

	var dst string
	switch len(cmd.Args) {
	case 1:
		user, err := user.Current()
		if err != nil {
			return cmdErrorf(cmd, "%s", err)
		}
		dst = user.HomeDir
	case 2:
		dst = cmd.Args[1]
		if dst == "-" {
			return cdPop(cmd)
		}
	default:
		fmt.Fprintln(cmd.Stderr, "Usage: cd [directory]")
		return 1
	}

	if cwd, err := os.Getwd(); err != nil {
		cmdErrorf(cmd, "%s", err)
	} else {
		dirStack.Push(cwd)
	}

	if err := os.Chdir(dst); err != nil {
		dirStack.Pop()
		return cmdErrorf(cmd, "%s", err)
	}
	return 0
}

func cdPop(cmd *exec.Cmd) uint8 {
	dst := dirStack.Pop()
	if dst == nil {
		return cmdErrorf(cmd, "the directory stack is empty")
	}

	if err := os.Chdir(*dst); err != nil {
		return cmdErrorf(cmd, "%s", err)
	}
	return 0
}

func cmdEcho(cmd *exec.Cmd, _ context) uint8 {
	// Cast to []any
	args := make([]any, len(cmd.Args)-1)
	for i := range args {
		args[i] = cmd.Args[i+1]
	}

	fmt.Fprintln(cmd.Stdout, args...)
	return 0
}

func cmdEval(cmd *exec.Cmd, _ context) uint8 {
	cmd.Args = shiftDashDash(cmd.Args)
	if len(cmd.Args) == 1 {
		cmd.Args = append(cmd.Args, "-")
	}
	for _, f := range cmd.Args[1:] {
		var (
			buf []byte
			err error
		)

		if f == "-" {
			buf, err = io.ReadAll(cmd.Stdin)
		} else {
			buf, err = os.ReadFile(f)
		}

		if err != nil {
			return cmdErrorf(cmd, "%s", err)
		}

		l := newLexer(string(buf))
		p := newParser(l.out)
		go l.run()
		globalVm.run(p.run())
	}
	return 0
}

func cmdExec(cmd *exec.Cmd, _ context) uint8 {
	var (
		zflag  bool
		zeroth string
	)

	usage := func() uint8 {
		fmt.Fprintln(cmd.Stderr, "Usage: exec [-z argument] command [argument ...]")
		return 1
	}

	flags, optind, err := opts.GetLong(cmd.Args, []opts.LongOpt{
		{Short: 'z', Long: "zero", Arg: opts.Required},
	})
	if err != nil {
		cmdErrorf(cmd, "%s", err)
		return usage()
	}

	for _, f := range flags {
		switch f.Key {
		case 'z':
			zflag = true
			zeroth = f.Value
		}
	}

	rest := cmd.Args[optind:]
	if len(rest) == 0 {
		return usage()
	}

	argv0, err := exec.LookPath(rest[0])
	if err != nil && !errors.Is(err, exec.ErrDot) {
		return cmdErrorf(cmd, "unable to find ‘%s’ in $PATH", rest[0])
	}
	if zflag {
		rest[0] = zeroth
	}
	err = syscall.Exec(argv0, rest, cmd.Environ())
	return cmdErrorf(cmd, "failed to exec ‘%s’: %s", argv0, err)
}

func cmdExit(cmd *exec.Cmd, _ context) uint8 {
	cmd.Args = shiftDashDash(cmd.Args)
	lo, hi := 0, math.MaxUint8

	var n int
	if len(cmd.Args) > 1 {
		var err error
		s := cmd.Args[1]
		n, err = strconv.Atoi(s)
		switch {
		case errors.Is(err, strconv.ErrRange) || n < lo || n > hi:
			return cmdErrorf(cmd, "exit code ‘%s’ must be in the range %d–%d", s, lo, hi)
		case err != nil:
			return cmdErrorf(cmd, "‘%s’ isn’t a valid integer", s)
		}
	}

	os.Exit(n)
	panic("unreachable")
}

func cmdFalse(_ *exec.Cmd, _ context) uint8 {
	return 1
}

func cmdQuote(cmd *exec.Cmd, _ context) uint8 {
	cmd.Args = shiftDashDash(cmd.Args)
	for _, arg := range cmd.Args[1:] {
		s := "'#"
		for strings.Contains(arg, s) {
			s += string('#')
		}
		fmt.Fprintf(cmd.Stdout, "r%s'%s%s\n", s[1:], arg, s)
	}
	return 0
}

func cmdRead(cmd *exec.Cmd, ctx context) uint8 {
	var Dflag, gflag bool

	usage := func() uint8 {
		fmt.Fprintln(cmd.Stderr, "Usage: read [-Dg] [-n num] [-d string] variable")
		return 1
	}

	flags, optind, err := opts.GetLong(cmd.Args, []opts.LongOpt{
		{Short: 'd', Long: "delimiters", Arg: opts.Required},
		{Short: 'D', Long: "no-empty", Arg: opts.None},
		{Short: 'g', Long: "global", Arg: opts.None},
		{Short: 'n', Long: "count", Arg: opts.Required},
	})
	if err != nil {
		cmdErrorf(cmd, "%s", err)
		return usage()
	}

	var ds []byte
	cnt := math.MaxInt
	for _, f := range flags {
		switch f.Key {
		case 'd':
			ds = []byte(f.Value)
		case 'D':
			Dflag = true
		case 'g':
			gflag = true
		case 'n':
			n, err := strconv.Atoi(f.Value)
			if err != nil {
				cmdErrorf(cmd, "%s", err)
				return usage()
			}
			cnt = n
		}
	}

	rest := cmd.Args[optind:]
	if len(rest) != 1 {
		return usage()
	}

	sb := strings.Builder{}
	buf := make([]byte, 1)
	parts := []string{}
outer:
	for cnt > 0 {
		_, err := cmd.Stdin.Read(buf)
		switch {
		case errors.Is(err, io.EOF):
			if sb.Len() > 0 {
				parts = append(parts, sb.String())
			}
			break outer
		case err != nil:
			return cmdErrorf(cmd, "%s", err)
		}

		b := buf[0]
		if bytes.IndexByte(ds, b) != -1 {
			cnt--
			parts = append(parts, sb.String())
			sb.Reset()
		} else {
			sb.WriteByte(buf[0])
		}
	}

	if Dflag {
		parts = slices.DeleteFunc(parts, func(s string) bool {
			return s == ""
		})
	}

	if len(parts) > 0 {
		p := parts[len(parts)-1]
		if n := len(p); n > 0 && p[n-1] == '\n' {
			p = p[:n-1]
		}
	}

	ident := rest[0]
	cmd.Args = cmd.Args[0:1]
	if gflag {
		cmd.Args = append(cmd.Args, "-g")
	}
	cmd.Args = append(cmd.Args, ident)
	cmd.Args = append(cmd.Args, parts...)
	res := cmdSet(cmd, ctx)
	if len(parts) == 0 {
		return 1
	}
	return res
}

func cmdSet(cmd *exec.Cmd, ctx context) uint8 {
	var eflag, gflag bool
	scope := ctx.scope

	usage := func() uint8 {
		fmt.Fprintln(cmd.Stderr, "Usage: set [-g] variable [value ...]\n"+
			"       set -e variable [value]")
		return 1
	}

	flags, optind, err := opts.GetLong(cmd.Args, []opts.LongOpt{
		{Short: 'e', Long: "environment", Arg: opts.None},
		{Short: 'g', Long: "global", Arg: opts.None},
	})
	if err != nil {
		cmdErrorf(cmd, "%s", err)
		return usage()
	}

	for _, f := range flags {
		switch f.Key {
		case 'e':
			eflag = true
		case 'g':
			gflag = true
		}
	}

	if gflag || ctx.scope == nil {
		scope = globalVariableMap
	}

	rest := cmd.Args[optind:]
	if len(rest) == 0 || eflag && len(rest) > 2 || eflag && gflag {
		return usage()
	}

	ident := rest[0]
	if slices.Contains(reservedNames, ident) {
		return cmdErrorf(cmd, "the ‘%s’ variable is read-only", ident)
	}
	if ok, r := isRefName(ident); !ok {
		return cmdErrorf(cmd, "rune ‘%c’ is not allowed in variable names", r)
	}

	switch {
	case eflag && len(rest) == 1:
		if err := os.Unsetenv(ident); err != nil {
			return cmdErrorf(cmd, "%s", err)
		}
	case eflag:
		if err := os.Setenv(ident, rest[1]); err != nil {
			return cmdErrorf(cmd, "%s", err)
		}
	case len(rest) == 1:
		delete(scope, ident)
	default:
		scope[ident] = rest[1:]
	}

	return 0
}

func cmdTrue(_ *exec.Cmd, _ context) uint8 {
	return 0
}

func cmdType(cmd *exec.Cmd, _ context) uint8 {
	cmd.Args = shiftDashDash(cmd.Args)
	if len(cmd.Args) < 2 {
		fmt.Fprintln(cmd.Stderr, "Usage: type identifier ...")
		return 1
	}

	for _, a := range cmd.Args[1:] {
		if _, ok := globalFuncMap[a]; ok {
			fmt.Fprintln(cmd.Stdout, "function")
		} else if _, ok := builtins[a]; ok {
			fmt.Fprintln(cmd.Stdout, "builtin")
		} else if _, err := exec.LookPath(a); err == nil || errors.Is(err, exec.ErrDot) {
			fmt.Fprintln(cmd.Stdout, "executable")
		} else {
			fmt.Fprintln(cmd.Stdout, "unknown")
		}
	}

	return 0
}

func cmdUmask(cmd *exec.Cmd, _ context) uint8 {
	cmd.Args = shiftDashDash(cmd.Args)

	if len(cmd.Args) < 2 {
		u := syscall.Umask(022)
		syscall.Umask(u)
		fmt.Fprintf(cmd.Stdout, "%04o\n", u)
	} else {
		s := cmd.Args[1]
		u, err := strconv.ParseUint(s, 8, 0)
		switch {
		case errors.Is(err, strconv.ErrRange), err == nil && u < 0 || u > 0777:
			cmdErrorf(cmd, "umask ‘%s’ is outside the allowed range of [0, 0777]", s)
		case errors.Is(err, strconv.ErrSyntax):
			cmdErrorf(cmd, "‘%s’ isn’t a valid umask", s)
		}
		if err != nil {
			return 1
		}
		syscall.Umask(int(u))
	}
	return 0
}

func shiftDashDash(s []string) []string {
	if len(s) > 1 && s[1] == "--" {
		return s[1:]
	}
	return s
}

func cmdErrorf(cmd *exec.Cmd, format string, args ...any) uint8 {
	format = fmt.Sprintf("%s: %s\n", cmd.Args[0], format)
	fmt.Fprintf(cmd.Stderr, format, args...)
	return 1
}
