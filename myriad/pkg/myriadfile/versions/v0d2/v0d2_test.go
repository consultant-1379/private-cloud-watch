// Copyright 2016 Ericsson AB All Rights Reserved.

package v0d2

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var example = `
version = "v0.2"

job "nameOne" {
    command = "sleep 20"
}
`

func TestParse(t *testing.T) {
	p := &parser{}
	var str string
	var i interface{}
	var err error
	var jobs []Job
	var ok bool

	str = strings.TrimLeft(example, "\n")
	i, err = p.Parse([]byte(str))
	assert.Nil(t, err)
	jobs, ok = i.([]Job)
	assert.True(t, ok)
	assert.Len(t, jobs, 1)
	assert.Len(t, jobs[0].Command, 2)
	assert.Equal(t, jobs[0].Name, "nameOne")
	assert.False(t, jobs[0].WaitOn, false)
}
