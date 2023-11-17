// Package vars exists so that it can be imported by the ast package without
// causing an import cycle.  There is probably a better way to do this.

package vars

var VarTable map[string][]string = make(map[string][]string, 64)
