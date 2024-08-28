// Copyright 2016 Ericsson AB All Rights Reserved.

// v0d2 myriadfile consists of single lines per job where the
// first word is the job name and the remainder of the line is
// the job command.
package v0d2

import (
	"fmt"
	"path"

	"github.com/google/shlex"
	"github.com/hashicorp/hcl"

	"github.com/spf13/viper"

	"github.com/erixzone/myriad/pkg/myriadfile/versions"
)

const version = "v0.2"

type parser struct{}

func init() {
	p := &parser{}
	if err := versions.RegisterFormat(p, version, ""); err != nil {
		panic(err)
	}
}

type Input struct {
	Src string `hcl:"src"`
}

type Output struct {
	Dst string `hcl:"dst"`
}

type job struct {
	Logfile string            `hcl:"-"`
	Command string            `hcl:"command"`
	WaitOn  bool              `hcl:"wait"`
	Input   map[string]Input  `hcl:"input"`
	Output  map[string]Output `hcl:"output"`
}

type Job struct {
	Name    string
	Logfile string
	Command []string
	WaitOn  bool
	Input   map[string]Input
	Output  map[string]Output
}

type MF struct {
	Version string         `hcl:"version"`
	Jobs    map[string]job `hcl:"job"`
}

// Parse performs the parsing of io.Reader (a myriadfile) and
// returns the list of jobs or an error.
func (p *parser) Parse(b []byte) (interface{}, error) {
	mf := &MF{}
	if err := hcl.Unmarshal(b, mf); err != nil {
		return nil, err
	}
	var parsed []Job
	for name, job := range mf.Jobs {
		parsedCommand, err := shlex.Split(job.Command)
		if err != nil {
			return nil, err
		}

		njob := Job{
			Name:    name,
			Logfile: getJobLogfile(name),
			Command: parsedCommand,
			WaitOn:  job.WaitOn,
			Input:   job.Input,
			Output:  job.Output,
		}
		parsed = append(parsed, njob)
	}

	return parsed, nil
}

// Upgrade is not currently implemented for this format as it is
// the latest version.
func (p *parser) Upgrade(i interface{}) (interface{}, error) {
	return i, nil
}

func getJobLogfile(name string) string {
	out := viper.GetString("out")
	if out == "" {
		return out
	}
	return path.Join(out, fmt.Sprintf("%s-%s", name, "stdio"))
}
