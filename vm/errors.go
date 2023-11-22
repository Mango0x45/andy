package vm

import (
	"fmt"
	"math"
)

const shellExitCode = math.MaxUint8

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
	return shellExitCode
}

func (e errFileOp) Error() string {
	return fmt.Sprintf("Failed to %s file ‘%s’: %s", e.desc, e.file, e.err)
}

type errClobber struct {
	file string // File related to the error
}

func (e errClobber) ExitCode() uint8 {
	return shellExitCode
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
	return shellExitCode
}

func (e errInternal) Error() string {
	return e.e.Error()
}

type errExpected struct {
	want, got string
}

func (e errExpected) ExitCode() uint8 {
	return shellExitCode
}

func (e errExpected) Error() string {
	return fmt.Sprintf("Expected %s but got %s", e.want, e.got)
}

type shellError interface {
	isShellError()
}

func (_ errClobber) isShellError()  {}
func (_ errFileOp) isShellError()   {}
func (_ errInternal) isShellError() {}
func (_ errExpected) isShellError() {}
