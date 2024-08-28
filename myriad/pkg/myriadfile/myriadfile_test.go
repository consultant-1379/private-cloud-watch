// Copyright 2016 Ericsson AB All Rights Reserved.

package myriadfile

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func mf(s string) io.Reader {
	s = strings.TrimLeft(s, "\n")
	return bytes.NewReader([]byte(s))
}

func TestVersions(t *testing.T) {
	assert.Len(t, Versions(), 2)
	data := mf(`
nameOne sleep 30
nameTwo echo "hello world!"
`)
	jobs, err := Parse(data)
	assert.Nil(t, err)
	assert.Len(t, jobs, 2)

	data = mf(`
version = "v0.2"
`)

	jobs, err = Parse(data)
	assert.Nil(t, err)
	assert.Len(t, jobs, 0)

	data = mf(`
version = "v0.2"

job "nameOne" {
    command = "sleep 30"
}

job "nameTwo" {
    command = <<EOF
echo "hello world!"
EOF
    wait = true
}
`)

	jobs, err = Parse(data)
	assert.Nil(t, err)
	assert.Len(t, jobs, 2)
}

func TestBadVersions(t *testing.T) {
	data := mf(`
version = "0.2"

job "nameOne" {
	command = "sleep 30"
}
`)
	_, err := Parse(data)
	assert.NotNil(t, err)
}
