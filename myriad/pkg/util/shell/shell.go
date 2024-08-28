// Package shell impelments helper functions around os/exec
package shell

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/erixzone/myriad/pkg/util/log"
)

// TODO: shell -> exec implement exec.Cmd with logging + printing wrappers.
// TODO: shell -> implement wrappers around exec.Cmd

// StdoutLines returns the command's standard output split into lines.
func StdoutLines(command string, args ...string) (lines []string, err error) {
	cmd := exec.Command(command, args...)
	log.Debugf("Exec: %s %s\n", command, strings.Join(args, " "))
	cmd.Stderr = os.Stderr
	stdout, err := cmd.Output()
	if err != nil {
		return
	}
	sc := bufio.NewScanner(bytes.NewReader(stdout))
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	return lines, sc.Err()
}

// StdoutLine returns the first line from a function call or an error if the
// function failed or if multiple lines were returned.
func StdoutLine(command string, args ...string) (line string, err error) {
	lines, err := StdoutLines(command, args...)
	if err != nil {
		return
	}
	if len(lines) > 1 {
		err = fmt.Errorf("Caution: Multiple lines returned")
	} else if len(lines) == 0 {
		err = fmt.Errorf("No lines returned!")
	} else {
		line = lines[0]
	}
	return
}

// StdPipes just prints out everything that's happening during a command execution.
func StdPipes(command string, args ...string) *exec.Cmd {
	cmd := exec.Command(command, args...)
	log.WithField("Cmd", append([]string{cmd.Path}, cmd.Args...)).Debug("RunCommand")
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd
}

// UnmarshalJSON executes the command and tries to unmarshal the stdout into
// the provided interface. An error is returned if either the command or the
// unmarshal fails.
func UnmarshalJSON(cmd *exec.Cmd, i interface{}) (err error) {
	cmd.Stdout = nil // It may have been redirected, but we need it.
	e := log.WithField("Cmd", append([]string{cmd.Path}, cmd.Args...))
	e.Debug("RunCommand")
	j, err := cmd.Output()
	if err != nil {
		e.WithError(err).Error("RunCommand")
		return
	}
	return json.Unmarshal(j, i)
}
