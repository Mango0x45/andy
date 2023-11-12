package vm

import (
	"fmt"
	"math"
)

type commandResult interface {
	error
	ExitCode() uint8
}

type errFileOp struct {
	desc string // Attempted action on file (‘open’, ‘stat’, etc.)
	file string // File related to the error
	err  error  // Error that caused this
}

func (e errFileOp) ExitCode() uint8 {
	return math.MaxUint8
}

func (e errFileOp) Error() string {
	return fmt.Sprintf("Failed to %s file ‘%s’: %s", e.desc, e.file, e.err)
}

type errClobber struct {
	file string // File related to the error
}

func (e errClobber) ExitCode() uint8 {
	return math.MaxUint8
}

func (e errClobber) Error() string {
	return fmt.Sprintf("Won’t clobber file ‘%s’; did you mean to use ‘>!’?",
		e.file)
}

type errExitCode uint8

func (e errExitCode) ExitCode() uint8 {
	return uint8(e)
}

func (_ errExitCode) Error() string {
	return ""
}

type errInternal struct {
	e error
}

func (e errInternal) ExitCode() uint8 {
	return math.MaxUint8
}

func (e errInternal) Error() string {
	return e.e.Error()
}

type shellError interface {
	isShellError()
}

func (_ errClobber) isShellError()  {}
func (_ errFileOp) isShellError()   {}
func (_ errInternal) isShellError() {}
