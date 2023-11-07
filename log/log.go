package log

import (
	"fmt"
	"os"
)

var CrashOnError = false

// Err prints a diagnostic to the standard error according to format.  It also
// prepends the program name and appends a newline.  This is much like the
// errx(3) function from C unless CrashOnError is false in which case this will
// act like warnx(3).
func Err(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "andy: "+format+"\n", args...)

	if CrashOnError {
		os.Exit(1)
	}
}
