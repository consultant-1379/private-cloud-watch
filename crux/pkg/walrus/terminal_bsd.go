// +build darwin freebsd openbsd netbsd dragonfly
// +build !appengine

package walrus

import "syscall"

const ioctlReadTermios = syscall.TIOCGETA

// Termios is our name for termios.
type Termios syscall.Termios
