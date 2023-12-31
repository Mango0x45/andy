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
	"sync"
	"sync/atomic"
	"syscall"

	"git.sr.ht/~mango/andy/pkg/stack"
	"git.sr.ht/~mango/opts/v2"
)

type builtin func(cmd *exec.Cmd, ctx context) uint8

var (
	builtins      map[string]builtin
	dirStack      = stack.New[string](64)
	reservedNames = []string{"cdstack", "pid", "ppid", "status"}
)

var asyncProcs struct {
	wg  sync.WaitGroup
	wgs map[uint64]*sync.WaitGroup
	mtx sync.Mutex
	nId atomic.Uint64
}

func init() {
	builtins = map[string]builtin{
		"!":     cmdBang,
		"async": cmdAsync,
		"call":  cmdCall,
		"cd":    cmdCd,
		"echo":  cmdEcho,
		"eval":  cmdEval,
		"exec":  cmdExec,
		"exit":  cmdExit,
		"false": cmdFalse,
		"get":   cmdGet,
		"quote": cmdQuote,
		"read":  cmdRead,
		"set":   cmdSet,
		"true":  cmdTrue,
		"type":  cmdType,
		"umask": cmdUmask,
		"wait":  cmdWait,
	}
	asyncProcs.wgs = make(map[uint64]*sync.WaitGroup, 32)
}

func cmdBang(cmd *exec.Cmd, ctx context) uint8 {
	cmd.Args = shiftDashDash(cmd.Args)
	if len(cmd.Args) == 1 {
		fmt.Fprintln(cmd.Stderr, "Usage: ! command [argument ...]")
		return 1
	}
	cmd.Args = cmd.Args[1:]
	if res := execPreparedCommand(dupCmd(cmd), ctx); cmdFailed(res) {
		return 0
	}
	return 1
}

func cmdAsync(cmd *exec.Cmd, ctx context) uint8 {
	var ivar string
	usage := func() uint8 {
		fmt.Fprintln(cmd.Stderr, "Usage: async [-i [var]] command [argument ...]")
		return 1
	}
	flags, rest, err := opts.GetLong(cmd.Args, []opts.LongOpt{
		{Short: 'i', Long: "id", Arg: opts.Optional},
	})
	if err != nil {
		cmdErrorf(cmd, "%s", err)
		return usage()
	}
	if len(rest) == 0 {
		return usage()
	}

	var id uint64
	for _, f := range flags {
		switch f.Key {
		case 'i':
			if f.Value != "" {
				ivar = f.Value
			} else {
				ivar = "_"
			}
			id = asyncProcs.nId.Add(1)
		}
	}

	cmd.Args = rest
	asyncProcs.wg.Add(1)
	if id > 0 {
		asyncProcs.mtx.Lock()
		wg := &sync.WaitGroup{}
		wg.Add(1)
		asyncProcs.wgs[id] = wg
		asyncProcs.mtx.Unlock()
		// TODO: Assert ivar is a valid varref
		globalVariableMap[ivar] = []string{strconv.FormatUint(id, 10)}
	}
	go func() {
		defer asyncProcs.wg.Done()
		if id > 0 {
			defer func() {
				asyncProcs.mtx.Lock()
				asyncProcs.wgs[id].Done()
				delete(asyncProcs.wgs, id)
				asyncProcs.mtx.Unlock()
			}()
		}
		_ = execPreparedCommand(dupCmd(cmd), ctx)
	}()

	return 0
}

func cmdCall(cmd *exec.Cmd, ctx context) uint8 {
	var bflag, cflag bool
	usage := func() uint8 {
		fmt.Fprintln(cmd.Stderr, "Usage: call [-bc] command [argument ...]")
		return 1
	}

	flags, rest, err := opts.GetLong(cmd.Args, []opts.LongOpt{
		{Short: 'b', Long: "builtin", Arg: opts.None},
		{Short: 'c', Long: "command", Arg: opts.None},
	})
	if len(rest) == 0 {
		return usage()
	}
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

	if !bflag && !cflag {
		bflag = true
		cflag = true
	}

	if bflag {
		if b, ok := builtins[rest[0]]; ok {
			cmd.Args = rest
			return b(cmd, ctx)
		}
	}

	if cflag {
		cmd.Args = rest
		c := dupCmd(cmd)
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
	if dst, ok := dirStack.Pop(); !ok {
		return cmdErrorf(cmd, "the directory stack is empty")
	} else if err := os.Chdir(dst); err != nil {
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

func cmdEval(cmd *exec.Cmd, ctx context) uint8 {
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
		execTopLevels(p.run(), ctx)
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

	flags, rest, err := opts.GetLong(cmd.Args, []opts.LongOpt{
		{Short: 'z', Long: "zero", Arg: opts.Required},
	})
	if len(rest) == 0 {
		return usage()
	}
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

func cmdGet(cmd *exec.Cmd, ctx context) uint8 {
	var dflag, eflag, gflag bool
	itemD, varD := "\n", "\n"
	scope := ctx.scope

	usage := func() uint8 {
		fmt.Fprintln(cmd.Stderr, "Usage: get [-g] [-Dd string] variable ...\n"+
			"       get -e [-D string] variable ...")
		return 1
	}

	flags, rest, err := opts.GetLong(cmd.Args, []opts.LongOpt{
		{Short: 'D', Long: "var-delimiter", Arg: opts.Required},
		{Short: 'd', Long: "item-delimiter", Arg: opts.Required},
		{Short: 'e', Long: "environment", Arg: opts.None},
		{Short: 'g', Long: "global", Arg: opts.None},
	})
	if len(rest) == 0 {
		return usage()
	}
	if err != nil {
		cmdErrorf(cmd, "%s", err)
		return usage()
	}

	for _, f := range flags {
		switch f.Key {
		case 'D':
			varD = f.Value
		case 'd':
			itemD = f.Value
			dflag = true
		case 'e':
			eflag = true
		case 'g':
			gflag = true
		}
	}

	if eflag && (dflag || gflag) {
		return usage()
	}

	if gflag || ctx.scope == nil {
		scope = globalVariableMap
	}

	for _, a := range rest {
		if ok, r := isRefName(a); !ok {
			return cmdErrorf(cmd, "rune ‘%c’ is not allowed in variable names", r)
		}
	}

	for i, a := range rest {
		if eflag {
			fmt.Fprint(cmd.Stdout, os.Getenv(a))
		} else {
			xs := scope[a]
			for i, s := range xs {
				fmt.Fprint(cmd.Stdout, s)
				if i < len(xs)-1 {
					fmt.Fprint(cmd.Stdout, itemD)
				}
			}
		}
		if i < len(rest)-1 {
			fmt.Fprint(cmd.Stdout, varD)
		}
	}
	fmt.Fprint(cmd.Stdout, "\n")

	return 0
}

func cmdQuote(cmd *exec.Cmd, _ context) uint8 {
	delim := "\n"
	usage := func() uint8 {
		fmt.Fprintln(cmd.Stderr, "Usage: quote [-d string] variable ...")
		return 1
	}

	flags, rest, err := opts.GetLong(cmd.Args, []opts.LongOpt{
		{Short: 'd', Long: "delimiter", Arg: opts.Required},
	})
	if err != nil {
		cmdErrorf(cmd, "%s", err)
		return usage()
	}
	if len(rest) < 1 {
		return usage()
	}

	for _, f := range flags {
		switch f.Key {
		case 'd':
			delim = f.Value
		}
	}

	for i, arg := range rest {
		s := "'#"
		for strings.Contains(arg, s) {
			s += string('#')
		}
		fmt.Fprintf(cmd.Stdout, "r%s'%s%s", s[1:], arg, s)
		if i < len(rest)-1 {
			fmt.Fprint(cmd.Stdout, delim)
		}
	}
	fmt.Fprint(cmd.Stdout, "\n")
	return 0
}

func cmdRead(cmd *exec.Cmd, ctx context) uint8 {
	var Dflag, gflag bool

	usage := func() uint8 {
		fmt.Fprintln(cmd.Stderr, "Usage: read [-Dg] [-d string] [-n num] variable")
		return 1
	}

	flags, rest, err := opts.GetLong(cmd.Args, []opts.LongOpt{
		{Short: 'D', Long: "no-empty", Arg: opts.None},
		{Short: 'd', Long: "delimiters", Arg: opts.Required},
		{Short: 'g', Long: "global", Arg: opts.None},
		{Short: 'n', Long: "count", Arg: opts.Required},
	})
	if len(rest) != 1 {
		return usage()
	}
	if err != nil {
		cmdErrorf(cmd, "%s", err)
		return usage()
	}

	var ds []byte
	cnt := math.MaxInt
	for _, f := range flags {
		switch f.Key {
		case 'D':
			Dflag = true
		case 'd':
			ds = []byte(f.Value)
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

	flags, rest, err := opts.GetLong(cmd.Args, []opts.LongOpt{
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

func cmdWait(cmd *exec.Cmd, ctx context) uint8 {
	cmd.Args = shiftDashDash(cmd.Args)

	if len(cmd.Args) <= 1 {
		asyncProcs.wg.Wait()
	} else {
		ids := make([]uint64, len(cmd.Args)-1)
		for i, a := range cmd.Args[1:] {
			n, err := strconv.ParseUint(a, 10, 64)
			if err != nil {
				return cmdErrorf(cmd, "%s", err)
			}
			ids[i] = n
		}

		for _, id := range ids {
			asyncProcs.mtx.Lock()
			wg, ok := asyncProcs.wgs[id]
			asyncProcs.mtx.Unlock()
			if ok {
				wg.Wait()
			}
		}
	}
	return 0
}

func dupCmd(cmd *exec.Cmd) *exec.Cmd {
	c := exec.Command(cmd.Args[0], cmd.Args[1:]...)
	c.Stdin = cmd.Stdin
	c.Stdout = cmd.Stdout
	c.Stderr = cmd.Stderr
	c.ExtraFiles = cmd.ExtraFiles
	return c
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
