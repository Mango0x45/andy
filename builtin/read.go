package builtin

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"os/exec"
	"slices"
	"strconv"
	"strings"

	"git.sr.ht/~mango/opts"
)

func read(cmd *exec.Cmd) uint8 {
	flags, optind, err := opts.GetLong(cmd.Args, []opts.LongOpt{
		{Short: 'd', Long: "delimiters", Arg: opts.Required},
		{Short: 'D', Long: "no-empty", Arg: opts.None},
		{Short: 'n', Long: "count", Arg: opts.Required},
	})
	if err != nil {
		errorf(cmd, "%s", err)
		return readUsage(cmd)
	}

	var ds []byte
	var noEmpty bool
	cnt := math.MaxInt
	for _, f := range flags {
		switch f.Key {
		case 'd':
			ds = []byte(f.Value)
		case 'D':
			noEmpty = true
		case 'n':
			n, err := strconv.Atoi(f.Value)
			if err != nil {
				errorf(cmd, "%s", err)
				return readUsage(cmd)
			}
			cnt = n
		}
	}

	cmd.Args = cmd.Args[optind:]
	if len(cmd.Args) != 1 {
		return readUsage(cmd)
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
			errorf(cmd, "%s", err)
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

	if noEmpty {
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

	cmd.Args = append([]string{"read"}, cmd.Args[0])
	cmd.Args = append(cmd.Args, parts...)
	res := set(cmd)
	if len(parts) == 0 {
		return 1
	}
	return res
}

func readUsage(cmd *exec.Cmd) uint8 {
	fmt.Fprintln(cmd.Stderr, "Usage: read [-D] [-n num] [-d string] variable")
	return 1
}
