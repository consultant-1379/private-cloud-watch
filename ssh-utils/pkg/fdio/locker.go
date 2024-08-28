package fdio

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

type LockFile os.File

func FileLock(name string) (*LockFile, error) {
	name, err := filepath.Abs(name)
	if err != nil {
		return nil, err
	}
	name += ".LOCK"
	for {
		fp, err := os.OpenFile(name, os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return nil, err
		}
		err = syscall.Flock(int(fp.Fd()), syscall.LOCK_EX)
		if err != nil {
			fp.Close()
			return nil, err
		}
		var nfi, ofi os.FileInfo
		nfi, err = os.Stat(name)
		if err == nil {
			ofi, err = fp.Stat()
		}
		if err == nil && os.SameFile(nfi, ofi) {
			fp.WriteString(fmt.Sprintf("%d\n", os.Getpid()))
			return (*LockFile)(fp), nil
		}
		fp.Close()
	}
}

func (t *LockFile) Unlock() {
	os.Remove((*os.File)(t).Name())
	(*os.File)(t).Close()
}
