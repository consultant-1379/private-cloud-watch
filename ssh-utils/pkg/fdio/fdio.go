package fdio

import (
	"io"
	"syscall"
)

type FdReader int

func (r FdReader) Read(p []byte) (int, error) {
	return syscall.Read(int(r), p)
}

func (r FdReader) Close() error {
	return syscall.Close(int(r))
}

func (r FdReader) Fd() int {
	return int(r)
}

type FdWriter int

func (w FdWriter) Write(p []byte) (int, error) {
	return syscall.Write(int(w), p)
}

func (w FdWriter) Close() error {
	return syscall.Close(int(w))
}

func (w FdWriter) Fd() int {
	return int(w)
}

type NonZeroReader struct {
	io.ReadCloser
}

func (r NonZeroReader) Read(p []byte) (int, error) {
	n, err := r.ReadCloser.Read(p)
	if n <= 0 {
		return 0, io.EOF
	}
	return n, err
}
