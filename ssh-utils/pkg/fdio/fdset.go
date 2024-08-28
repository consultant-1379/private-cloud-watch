package fdio

import (
	"syscall"
)

type FdSet struct {
	syscall.FdSet
}

func (p *FdSet) Set(fds ...int) {
	for _, fd := range fds {
		p.Bits[fd/64] |= 1 << (uint(fd) % 64)
	}
}

func (p *FdSet) IsSet(fd int) bool {
	return (p.Bits[fd/64] & (1 << (uint(fd) % 64))) != 0
}

func (p *FdSet) Zero() {
	for i := range p.Bits {
		p.Bits[i] = 0
	}
}
