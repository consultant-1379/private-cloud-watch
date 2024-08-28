package fdio

import (
	"io"
	"syscall"
)

type HupReader struct {
	fd   int
	hup  [2]int
	rfds FdSet
	nfds int
}

func NewHupReader(fd int) (*HupReader, error) {
	h := new(HupReader)
	h.fd = fd
	err := syscall.Pipe(h.hup[:])
	if err != nil {
		return nil, err
	}
	h.nfds = h.hup[0]
	if h.fd > h.nfds {
		h.nfds = h.fd
	}
	h.nfds++

	return h, nil
}

func (h *HupReader) Read(p []byte) (int, error) {
	if h.hup[0] < 0 {
		return 0, io.EOF
	}
	h.rfds.Set(h.fd, h.hup[0])
	_, err := syscall.Select(h.nfds, &h.rfds.FdSet, nil, nil, nil)
	if err != nil {
		return 0, err
	}
	if h.rfds.IsSet(h.hup[0]) {
		return h.sendErr(0, io.EOF)
	}
	n, err := syscall.Read(h.fd, p)
	if err != nil {
		return h.sendErr(n, err)
	}
	if n == 0 {
		return h.sendErr(0, io.EOF)
	}
	return n, nil
}

func (h *HupReader) sendErr(n int, err error) (int, error) {
	if h.hup[0] >= 0 {
		syscall.Close(h.hup[0])
		h.hup[0] = -1
	}
	return n, err
}

func (h *HupReader) Hangup() {
	if h.hup[1] >= 0 {
		syscall.Write(h.hup[1], []byte{0})
		syscall.Close(h.hup[1])
		h.hup[1] = -1
	}
}

func (h *HupReader) Close() error {
	h.Hangup()
	return syscall.Close(h.fd)
}

func (h *HupReader) Fd() int {
	return h.fd
}
