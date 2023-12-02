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

	"git.sr.ht/~mango/opts"
)

type builtin func(cmd *exec.Cmd, ctx context) uint8

type stack []string

var (
	builtins      map[string]builtin
	dirStack      stack = make([]string, 0, 64)
	reservedNames []string = []string{"pid", "status"}
)

func init() {
	builtins = map[string]builtin{
		".":     cmdDot,
		"cd":    cmdCd,
		"cmd":   cmdCmd,
		"echo":  cmdEcho,
		"false": cmdFalse,
		"read":  cmdRead,
		"set":   cmdSet,
		"true":  cmdTrue,
	}
}

func (s *stack) push(dir string) {
	*s = append(*s, dir)
}

func (s *stack) pop() (string, bool) {
	if len(*s) == 0 {
		return "", false
	}
	n := len(*s) - 1
	d := (*s)[n]
	*s = (*s)[:n]
	return d, true
}

func cmdDot(cmd *exec.Cmd, _ context) uint8 {
	if len(cmd.Args) == 1 {
		cmd.Args = append(cmd.Args, "-")
	}
	for _, f := range cmd.Args[1:] {
		var buf []byte
		var err error

		if f == "-" {
			buf, err = io.ReadAll(cmd.Stdin)
		} else {
			buf, err = os.ReadFile(f)
		}

		if err != nil {
			cmdErrorf(cmd, "%s", err)
			return 1
		}

		l := newLexer(string(buf))
		p := newParser(l.out)
		go l.run()
		globalVm.run(p.run())
	}
	return 0
}

func cmdCd(cmd *exec.Cmd, _ context) uint8 {
	var dst string
	switch len(cmd.Args) {
	case 1:
		user, err := user.Current()
		if err != nil {
			cmdErrorf(cmd, "%s", err)
			return 1
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
		dirStack.push(cwd)
	}

	if err := os.Chdir(dst); err != nil {
		dirStack.pop()
		cmdErrorf(cmd, "%s", err)
		return 1
	}
	return 0
}

func cdPop(cmd *exec.Cmd) uint8 {
	dst, ok := dirStack.pop()
	if !ok {
		cmdErrorf(cmd, "the directory stack is empty")
		return 1
	}

	if err := os.Chdir(dst); err != nil {
		cmdErrorf(cmd, "%s", err)
		return 1
	}
	return 0
}

func cmdCmd(cmd *exec.Cmd, _ context) uint8 {
	if len(cmd.Args) < 2 {
		fmt.Fprintln(cmd.Stderr, "Usage: cmd command [args ...]")
		return 1
	}

	c := exec.Command(cmd.Args[1], cmd.Args[2:]...)
	c.Stdin, c.Stdout, c.Stderr = cmd.Stdin, cmd.Stdout, cmd.Stderr
	c.ExtraFiles = cmd.ExtraFiles
	err := c.Run()
	code := c.ProcessState.ExitCode()

	if err != nil && code == -1 {
		cmdErrorf(cmd, "%s", err)
		return 1
	}
	return uint8(code)
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

func cmdFalse(_ *exec.Cmd, _ context) uint8 {
	return 1
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
			cmdErrorf(cmd, "%s", err)
			return 1
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
		cmdErrorf(cmd, "the ‘%s’ variable is read-only", ident)
		return 1
	}
	if ok, r := isRefName(ident); !ok {
		cmdErrorf(cmd, "rune ‘%c’ is not allowed in variable names", r)
		return 1
	}

	switch {
	case eflag && len(rest) == 1:
		if err := os.Unsetenv(ident); err != nil {
			cmdErrorf(cmd, "%s", err)
			return 1
		}
	case eflag:
		if err := os.Setenv(ident, rest[1]); err != nil {
			cmdErrorf(cmd, "%s", err)
			return 1
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

func cmdErrorf(cmd *exec.Cmd, format string, args ...any) {
	format = fmt.Sprintf("%s: %s\n", cmd.Args[0], format)
	fmt.Fprintf(cmd.Stderr, format, args...)
}
