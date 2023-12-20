//go:build !(linux || darwin)

package main

import "os"

var signals = map[string]os.Signal{}
