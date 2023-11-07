package vm

import "os"

type streams struct {
	in, out, err *os.File
}
