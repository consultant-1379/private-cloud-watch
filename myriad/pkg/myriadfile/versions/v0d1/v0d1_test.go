// Copyright 2016 Ericsson AB All Rights Reserved.

package v0d1

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/erixzone/myriad/pkg/myriadfile/versions/v0d2"
)

func TestParse(t *testing.T) {
	p := &parser{}
	var str string
	var i interface{}
	var err error
	var jobs []job
	var ok bool

	str = `
nameOne sleep 20
nameTwo sleep 30
`
	str = strings.TrimLeft(str, "\n")
	i, err = p.Parse([]byte(str))
	assert.Nil(t, err)
	jobs, ok = i.([]job)
	assert.True(t, ok)
	assert.Len(t, jobs, 2)
	assert.Len(t, jobs[0].Command, 2)
	assert.Equal(t, jobs[0].Name, "nameOne")
	assert.False(t, jobs[0].WaitOn, false)
	// Expect the last job in the set to always have WaitOn = true
	assert.True(t, jobs[1].WaitOn)

	str = `
nameOne echo "hello world!"
`
	str = strings.TrimLeft(str, "\n")
	i, err = p.Parse([]byte(str))
	assert.Nil(t, err)
	jobs, ok = i.([]job)
	assert.True(t, ok)
	assert.True(t, jobs[0].WaitOn)
	assert.Len(t, jobs[0].Command, 2)
}

func TestMarshal(t *testing.T) {
	var err error
	var str string
	var i interface{}

	p := &parser{}
	str = `
nameOne sleep 20
nameTwo sleep 30
`
	str = strings.TrimLeft(str, "\n")
	i, err = p.Parse([]byte(str))
	assert.Nil(t, err)
	b, err := p.Marshal(i)
	assert.Nil(t, err)
	assert.Equal(t, str, string(b))
}

func TestUpgrade(t *testing.T) {
	p := &parser{}
	var str string
	var i interface{}
	var err error
	var ok bool

	str = `
nameOne sleep 20
`
	str = strings.TrimLeft(str, "\n")
	i, err = p.Parse([]byte(str))
	assert.Nil(t, err)
	_, ok = i.([]job)
	assert.True(t, ok)
	i, err = p.Upgrade(i)
	assert.Nil(t, err)

	njobs, ok := i.([]v0d2.Job)
	assert.True(t, ok)
	assert.Len(t, njobs, 1)
	assert.Equal(t, njobs[0].Name, "nameOne")
	assert.Len(t, njobs[0].Command, 2)
}
