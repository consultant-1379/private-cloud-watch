package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/erixzone/xaas/platform/ssh-utils/pkg/fdio"
)

var helpFlag bool
var knownHosts string

func init() {
	flag.BoolVar(&helpFlag, "h", false, "show usage and exit")
	flag.StringVar(&knownHosts, "knownhosts", "", "known_hosts file (required)")
}

func usage(rc int) int {
	fmt.Fprintf(os.Stderr, "usage: %s [options] new-key-entry\n", os.Args[0])
	flag.PrintDefaults()
	return rc
}

func _main() int {
	flag.Parse()

	if helpFlag {
		return usage(0)
	}
	if knownHosts == "" || flag.NArg() != 1 {
		return usage(127)
	}

	newKey := flag.Arg(0) + "\n"
	addrLen := strings.Index(newKey, " ")
	if addrLen <= 0 {
		fmt.Fprintf(os.Stderr, "can't get host address from key entry\n")
		return 126
	}
	addrLen++ // include the space

	knownHosts, err := filepath.Abs(knownHosts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "can't get absolute path: %s\n", err)
		return 1
	}
	lock, err := fdio.FileLock(knownHosts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "can't get lock: %s\n", err)
		return 2
	}
	defer lock.Unlock()
	fout, err := ioutil.TempFile(filepath.Dir(knownHosts), "new-*-"+filepath.Base(knownHosts))
	if err != nil {
		fmt.Fprintf(os.Stderr, "can't create new file: %s\n", err)
		return 3
	}
	defer os.Remove(fout.Name()) // noop on success
	err = fout.Chmod(0600)
	if err != nil {
		fmt.Fprintf(os.Stderr, "can't chmod new file: %s\n", err)
		return 4
	}
	fp, err := os.Open(knownHosts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open failed: %s\n", err)
		return 5
	}
	defer fp.Close()
	fi, err := fp.Stat()
	if err != nil {
		fmt.Fprintf(os.Stderr, "can't stat existing file: %s\n", err)
		return 6
	}
	r := bufio.NewReader(fp)
	w := bufio.NewWriter(fout)
	for {
		l, err := r.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				fmt.Fprintf(os.Stderr, "read error: %s\n", err)
			}
			break
		}
		if addrLen > 0 && strings.HasPrefix(l, newKey[:addrLen]) {
			w.WriteString(newKey)
			addrLen = 0
		} else {
			w.WriteString(l)
		}
	}
	if addrLen > 0 {
		w.WriteString(newKey)
	}
	w.Flush()
	err = fout.Chmod(fi.Mode())
	if err != nil {
		fmt.Fprintf(os.Stderr, "can't re-chmod new file: %s\n", err)
	}
	fout.Close()
	err = os.Rename(fout.Name(), fp.Name())
	return 0
}

func main() {
	os.Exit(_main())
}
