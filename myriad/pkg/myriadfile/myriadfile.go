// Copyright 2016 Ericsson AB All Rights Reserved.

// Package myriadfile Parse myriad-files.
package myriadfile

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"regexp"

	"github.com/hashicorp/hcl"

	"github.com/erixzone/myriad/pkg/myriadfile/versions"
	"github.com/erixzone/myriad/pkg/util/log"

	// _ imports to register old versions of myriadfile as a side-effect.
	_ "github.com/erixzone/myriad/pkg/myriadfile/versions/v0d1" // v0.1

	// current version is referenced here so import it regularly
	"github.com/erixzone/myriad/pkg/myriadfile/versions/v0d2" // v0.2
)

const currentVersion = "v0.2"

// Job is the representation of a Job from the myriadfile that the drivers expect to use.
type Job v0d2.Job

type hclVersion struct {
	Version string `hcl:"version"`
}

// Parse takes an io.Reader and returns a list of jobs or an error.
func Parse(r io.Reader) ([]Job, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	// Attempt to extract version via HCL tag
	got := &hclVersion{}
	err = hcl.Decode(got, string(b))

	// May be a v0.1 version myriadfile, try that
	if err != nil && !looksLikeHCL(b) {
		log.Debug("Does not look like HCL, defaulting to v0.1")
		got = &hclVersion{Version: "v0.1"}
	} else if err != nil {
		return nil, err
	}

	// Parse the file into it's initial format.
	parser, err := versions.GetFileParser(got.Version)
	if err != nil {
		return nil, err
	}
	i, err := parser.Parse(b)
	if err != nil {
		return nil, err
	}
	if got.Version != currentVersion {
		// Then upgrade that format to the current version.
		fn, err := versions.Upgrade(got.Version, currentVersion)
		if err != nil {
			return nil, err
		}

		i, err = fn(i)
		if err != nil {
			return nil, err
		}
	}

	j, ok := i.([]v0d2.Job)
	if !ok {
		return nil, fmt.Errorf("Unknown error reading myriadfile")
	}
	var jobs []Job
	for _, k := range j {
		jobs = append(jobs, Job(k))
	}
	return jobs, nil
}

// Versions returns the list of versions that are registered.
func Versions() []string {
	return versions.Versions()
}

// Gross hack but attempt to see if the file contains a line '^version=%s$'
func looksLikeHCL(b []byte) bool {
	scan := bufio.NewReader(bytes.NewReader(b))
	re, err := regexp.Compile(`^\s*version\s*=\s*(\S+)\s*$`)
	if err != nil {
		panic(err)
	}

	var ln []byte
	for {
		ln, err = scan.ReadSlice('\n')
		if err != nil {
			break
		}
		parts := re.FindSubmatch(ln)
		if parts == nil || len(parts) == 0 {
			continue
		}
		log.Debugf("Looks like an invalid HCL file, found line: %q", string(parts[0]))
		return true
	}
	if err != io.EOF {
		panic(err)
	}
	return false
}
