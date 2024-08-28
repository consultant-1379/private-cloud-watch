// Copyright 2016 Ericsson AB All Rights Reserved.

// Package v0d1 defines the v0.1 definition of a myriadfile which
// consists of single lines per job where the first word is the
// job name and the remainder of the line is the job command.
// This package provides an upgrade path from v0.1 to v0.2.
package v0d1

import (
	"bufio"
	"bytes"
	"fmt"
	"path"
	"strings"

	"github.com/google/shlex"
	"github.com/spf13/viper"

	"github.com/erixzone/myriad/pkg/myriadfile/versions"
	"github.com/erixzone/myriad/pkg/myriadfile/versions/v0d2"
)

const version = "v0.1"

type parser struct{}

type job struct {
	Name    string
	Command []string
	Logfile string
	WaitOn  bool
}

// InvalidParse is returned from Parse if an error occurs
type InvalidParse struct {
	Text string
	Err  error
}

func (e InvalidParse) Error() string {
	ln := ""
	if e.Text != "" {
		ln = fmt.Sprintf("LINE: \"%s\"\n", e.Text)
	}
	errstr := ""
	if e.Err != nil {
		errstr = e.Err.Error()
	}
	return fmt.Sprintf("Error parsing text: %s%s", ln, errstr)
}

func init() {
	p := &parser{}
	versions.RegisterFormat(p, version, "v0.2")
}

// Parse performs the parsing of a byte array (a myriadfile) and
// returns the list of jobs or an error.
func (p *parser) Parse(b []byte) (interface{}, error) {
	f := bytes.NewReader(b)
	scanner := bufio.NewScanner(f)
	seen := make(map[string]bool)
	var parsed []job
	for scanner.Scan() {
		line := scanner.Text()
		fields, err := shlex.Split(line)
		if err != nil {
			return nil, err
		}
		if len(fields) > 1 {
			if _, ok := seen[fields[0]]; ok {
				xerr := fmt.Errorf("Repeated job name '%s'", fields[0])
				return nil, InvalidParse{Text: line, Err: xerr}
			}
			job := job{
				Name:    fields[0],
				Command: fields[1:],
				Logfile: getJobLogfile(fields[0]),
			}
			parsed = append(parsed, job)
			seen[fields[0]] = true
		} else {
			return nil, InvalidParse{Text: line}
		}
	}

	if len(parsed) > 0 {
		parsed[len(parsed)-1].WaitOn = true
	}

	if err := scanner.Err(); err != nil {
		return nil, InvalidParse{Err: err}
	}
	return parsed, nil
}

// Upgrade transforms the myriadfile to v0.2.
func (p *parser) Upgrade(i interface{}) (interface{}, error) {
	current, ok := i.([]job)
	if !ok {
		return nil, fmt.Errorf("Unknown type in unmarshal")
	}
	var up []v0d2.Job
	for _, job := range current {
		u := v0d2.Job{
			Name:    job.Name,
			Logfile: job.Logfile,
			Command: job.Command,
			WaitOn:  job.WaitOn,
		}
		up = append(up, u)
	}
	return up, nil
}

// Marshal returns a byte-array representation of the parsed myriadfile.
func (p *parser) Marshal(i interface{}) ([]byte, error) {
	jobs, ok := i.([]job)
	if !ok {
		return nil, fmt.Errorf("Unknown type in unmarshal")
	}
	var lines []string
	for _, job := range jobs {
		parts := []string{job.Name}
		parts = append(parts, job.Command...)
		lines = append(lines, strings.Join(parts, " "))
	}
	final := strings.Join(lines, "\n") + "\n"
	return []byte(final), nil
}

func getJobLogfile(name string) string {
	out := viper.GetString("out")
	if out == "" {
		return out
	}
	return path.Join(out, fmt.Sprintf("%s-%s", name, "stdio"))
}
