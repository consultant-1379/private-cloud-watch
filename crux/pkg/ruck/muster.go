package ruck

import (
	"io/ioutil"
	"strings"

	"github.com/erixzone/crux/pkg/clog"
	"github.com/erixzone/crux/pkg/crux"
)

var muster struct {
	init bool
	fmap map[string]string // index by filename to get hash
	smap map[string]string // index by symbol to get hash
}

// ReadMuster reads in the symbol table
func ReadMuster(logger clog.Logger) (string, *crux.Err) {
	file := "/tmp/cache/symtab"
	dir := "/tmp/cache/"

	muster.init = true
	muster.fmap = make(map[string]string)
	muster.smap = make(map[string]string)
	xx, err := ioutil.ReadFile(file)
	if err != nil {
		return dir, crux.ErrE(err)
	}
	for _, line := range strings.Split(strings.TrimSuffix(string(xx), "\n"), "\n") {
		fields := strings.Split(line, " ")
		if fields[0][0:1] == "@" {
			muster.fmap[fields[0][1:]] = dir + fields[1]
		} else {
			muster.smap[fields[0]] = dir + fields[1]
		}
	}
	if true {
		for k, v := range muster.fmap {
			logger.Log(nil, "muster file: %s -> %s", k, v)
		}
		for k, v := range muster.smap {
			logger.Log(nil, "muster sym: %s -> %s", k, v)
		}
	}
	return dir, nil
}
