# andy — a better yet simple shell

Andy is a shell named after my beloved OOP professor from my first year in
university, Andy Zaidman.  Andy if you’re reading this, you’re great!  It’s just
a shame that what you taught me best is why I should never use Java for
anything.

The goal of the Andy shell is to provide a more sane experience to the shell as
opposed to the highly deficient POSIX shells such as Bash and Zsh, while still
remaining simple and small unlike Fish.  It draws inspiration from various
shells, most notably Rc from the Plan 9 operating system.

This shell is very much a work in progress, don’t expect anything from this for
a while.  The currently implemented features are:

- [X] Simple commands (`cat foo bar`)
- [X] File reading (`<`)
- [X] File writing (`>`)
- [X] File appending (`>>`)
- [X] File clobbering (`>!`)
- [X] Read from /dev/null (`<_`)
- [X] Write to /dev/null (`>_`)
- [X] Pipelines (`cmd1 | … | cmdN`)
- [X] Condition chains (`cmd1 && … || cmdN`)
- [X] `cd` builtin function with `pushd/popd` behaviour
- [X] `echo` builtin function
- [X] `true` builtin function
- [X] `false` builtin function
- [X] Value lists (`(a b …)`)
- [X] String concationation (`foo'bar'"baz"`)
- [X] Cartesian product list concationation (`(foo bar).c; (a b)(c d)`)
- [X] Compound commands (`{ cmd; cmd }`)
- [X] If(-else) expressions (`if cmd { … } else if cmd { … } else { … }`)
- [X] While expressions (`while cmd { … }`)
- [X] Tilde expansion (`echo ~ ~username`)
- [X] Setting- and reading variables (`set x foo; set xʹ $x.c; echo $xʹ`)
- [X] Flattening variables (`$^var`)
- [X] Get variable length (`$#var`)
- [X] Index into variables (`$var[1 1 -2 5]`)
- [X] Process substitution (``​`{…}``, `<{…}`, `>{…}`, `<>{…}`)

## Example

This is *very* early days and *very* likely to change.  Also some of it is
already outdated.

```andy
# Define a function ‘greet’ that takes an argument ‘name’
fn greet name
    echo "Hello $name!"

# Multiline pipelines without newline escaping
grep foo /some/path
| sort
| nl

# Conditionals and writing to stderr
if test $# -lt 1 {
    echo "Usage: $0 [-f] pattern [file ...]" >!
    exit 1
}

# Read and write to and from /dev/null
cat <_
cat >_

# Explicit file clobbering with ‘>|’
echo hello >my-file
echo hello >my-file   # Fails
echo hello >|my-file  # Succeeds

# Pattern match strings, with implicit breaks and explicit fallthroughs
x = 123
switch $x {
case foo bar
    echo '$x is either ‘foo’ or ‘bar’'
case [0-9]*
    echo '$x starts with a number'
    fallthough
case *
    echo 'This is a default case'
}

# Lists
echo (foo bar).c  # Echos ‘foo.c bar.c’
cat (a b)(x y z)  # Reads the files ax, ay, az, bx, by, and bz

# The same
xs = 1 2 3
xs = (1 2 3)
xs = 1 (2 3)

# The same
xs = foo
xs = (foo)

xs = ('foo bar' baz)
cat $xs   # Read the files ‘foo bar’ and ‘baz’
cat $^xs  # Read the file ‘foo bar baz’

# Process substitution
diff <(cmd1) <(cmd2)
```
