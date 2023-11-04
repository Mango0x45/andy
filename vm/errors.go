package vm

import "fmt"

type errWontClobber struct {
	filename  string
	appending bool
}

func (e errWontClobber) Error() string {
	var op string
	if e.appending {
		op = ">>|"
	} else {
		op = ">|"
	}

	return fmt.Sprintf("File ‘%s’ exists; use the ‘%s’ operator to overwrite it",
		e.filename, op)
}

type errMultipleStrings []string

func (e errMultipleStrings) Error() string {
	return fmt.Sprintf("Expected a single string ‘%s’, but got the strings ‘%s’",
		e[0], e[1:])
}
