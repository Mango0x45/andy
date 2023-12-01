package main

import (
	"fmt"
	"math"
)

const cmdFailCode = math.MaxUint8

type commandResult interface {
	error
	ExitCode() uint8
}

type errFileOp struct {
	desc string
	file string
	err  error
}

func (e errFileOp) Error() string {
	return fmt.Sprintf("Failed to %s file ‘%s’: %s", e.desc, e.file, e.err)
}

type errClobber struct {
	file string
}

func (e errClobber) Error() string {
	return fmt.Sprintf("Won’t clobber file ‘%s’; did you mean to use ‘>!’?",
		e.file)
}

type errExitCode uint8

func (_ errExitCode) Error() string {
	return ""
}

type errInternal struct {
	e error
}

func (e errInternal) Error() string {
	return e.e.Error()
}

type errExpected struct {
	want, got any
}

func (e errExpected) Error() string {
	return fmt.Sprintf("Expected %s but got %s", e.want, e.got)
}

type errUnsupported string

func (e errUnsupported) Error() string {
	return fmt.Sprintf("Attempt to %s is unsupported", string(e))
}

func (e errClobber) ExitCode() uint8     { return cmdFailCode }
func (e errExpected) ExitCode() uint8    { return cmdFailCode }
func (e errFileOp) ExitCode() uint8      { return cmdFailCode }
func (e errInternal) ExitCode() uint8    { return cmdFailCode }
func (e errUnsupported) ExitCode() uint8 { return cmdFailCode }
func (e errExitCode) ExitCode() uint8    { return uint8(e) }

type shellError interface {
	isShellError()
}

func (_ errClobber) isShellError()     {}
func (_ errExpected) isShellError()    {}
func (_ errFileOp) isShellError()      {}
func (_ errInternal) isShellError()    {}
func (_ errUnsupported) isShellError() {}

func cmdFailed(e commandResult) bool {
	return e.ExitCode() != 0
}
