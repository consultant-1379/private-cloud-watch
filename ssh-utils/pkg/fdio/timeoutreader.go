package fdio

import (
	"errors"
	"syscall"
	"time"
)

type TimeoutReader struct {
	fd int
	tv syscall.Timeval
}

func NewTimeoutReader(fd int, t time.Duration) *TimeoutReader {
	h := new(TimeoutReader)
	h.fd = fd
	h.SetTimeout(t)
	return h
}

func (h *TimeoutReader) Fd() int {
	return h.fd
}

func (h *TimeoutReader) SetTimeout(t time.Duration) {
	h.tv = syscall.NsecToTimeval(t.Nanoseconds())
}

func (h *TimeoutReader) Read(p []byte) (int, error) {
	var fs FdSet
	fs.Set(h.fd)
	tv := h.tv
	_, err := syscall.Select(h.fd+1, &fs.FdSet, nil, nil, &tv)
	if err != nil {
		return 0, err
	}
	if !fs.IsSet(h.fd) {
		return 0, errors.New("timeout")
	}
	return syscall.Read(h.fd, p)
}

func (h *TimeoutReader) Close() error {
	return syscall.Close(h.fd)
}
