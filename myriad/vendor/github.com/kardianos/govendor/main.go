// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// vendor tool to copy external source code from GOPATH or remote location to the
// local vendor folder. See README.md for usage.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/kardianos/govendor/cliprompt"
	"github.com/kardianos/govendor/run"
)

func main() {
	prompt := &cliprompt.Prompt{}

	allArgs := os.Args

	if allArgs[len(allArgs)-1] == "-" {
		stdin := &bytes.Buffer{}
		if _, err := io.Copy(stdin, os.Stdin); err == nil {
			stdinArgs := strings.Fields(stdin.String())
			allArgs = append(allArgs[:len(allArgs)-1], stdinArgs...)
		}
	}

	msg, err := run.Run(os.Stdout, allArgs, prompt)
	if err == flag.ErrHelp {
		err = nil
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}
	msgText := ""
	switch msg {
	default:
		panic("Unknown message type")
	case run.MsgNone:
	case run.MsgFull:
		msgText = helpFull
	case run.MsgInit:
		msgText = helpInit
	case run.MsgList:
		msgText = helpList
	case run.MsgAdd:
		msgText = helpAdd
	case run.MsgUpdate:
		msgText = helpUpdate
	case run.MsgRemove:
		msgText = helpRemove
	case run.MsgFetch:
		msgText = helpFetch
	case run.MsgStatus:
		msgText = helpStatus
	case run.MsgSync:
		msgText = helpSync
	case run.MsgMigrate:
		msgText = helpMigrate
	case run.MsgGet:
		msgText = helpGet
	case run.MsgLicense:
		msgText = helpLicense
	case run.MsgGovendorLicense:
		msgText = msgGovendorLicenses
	}
	if len(msgText) > 0 {
		fmt.Fprint(os.Stderr, msgText)
	}
	if err != nil {
		os.Exit(2)
	}
	if msg != run.MsgNone {
		os.Exit(1)
	}
}
