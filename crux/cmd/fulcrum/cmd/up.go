// Copyright © 2016 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/erixzone/crux/pkg/crux"
)

type event struct {
	t  time.Time
	mi crux.MonInfo
}

type what struct {
	n      string
	fname  string
	uuid   string
	begin  time.Time
	latest time.Time
}

type evec struct {
	e []*event
	w map[string]*what
}

var debug bool
var e evec
var cursor time.Time
var cursorStep time.Duration

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "see who is up",
	Long:  `nada`,
	Run: func(cmd *cobra.Command, args []string) {
		e.w = make(map[string]*what, 6)
		vip := parseCmd(cmd)
		timeWindow := time.Duration(vip.GetInt("timer")) * time.Second
		cursorStep = time.Duration(vip.GetInt("step")) * time.Second
		fmt.Printf("tw=%s  cs=%s\n", timeWindow, cursorStep)
		// parse the input and send on to the windowing routine
		fs := bufio.NewScanner(os.Stdin)
		for fs.Scan() {
			txt := fs.Text()
			if strings.Index(txt, "focus=hb") < 0 {
				continue
			}
			if txt[0:5] != "time=" {
				continue
			}
			var e event
			xx := txt[6:]
			if i := strings.Index(xx, `"`); i >= 0 {
				tt, err := time.Parse(time.RFC3339, xx[0:i])
				if err != nil {
					fmt.Printf("skipping bad time (%s) in %s\n", xx[0:i], txt)
					continue
				}
				e.t = tt
			} else {
				fmt.Printf("skipping bad time format in %s\n", txt)
				continue
			}
			if cursor.IsZero() && !e.t.IsZero() {
				cursor = e.t.Add(-cursorStep).Round(cursorStep)
			}
			if i := strings.Index(txt, "msg="); i >= 0 {
				if j := strings.Index(txt[i:], `" focus=hb`); j >= 0 {
					if err := json.Unmarshal([]byte(txt[i+5:i+j]), &e.mi); err != nil {
						fmt.Printf("bad json for MonInfo in '%s'\n", txt[i+5:i+j])
						continue
					}
				} else {
					fmt.Printf("expected \" focus=hb in %s\n", txt[i:])
					continue
				}
			} else {
				fmt.Printf("expected a msg in %s\n", txt)
				continue
			}
			window(&e, timeWindow)
		}
		window(nil, timeWindow)
	},
}

func init() {
	upCmd.Flags().Int("timer", 5, "process lines by time window of this many seconds")
	upCmd.Flags().Int("step", 5, "move the time cursor by thsi many seconds")
	addCmd(upCmd)
}

// sort stuff
func (e evec) Len() int {
	return len(e.e)
}
func (e evec) Less(i, j int) bool {
	return e.e[i].mi.T.Before(e.e[j].mi.T)
}
func (e evec) Swap(i, j int) {
	e.e[i], e.e[j] = e.e[j], e.e[i]
}

func window(ev *event, win time.Duration) {
	var cutoff time.Time
	if ev == nil {
		// cutoff is the last timestamp
		cutoff = e.e[len(e.e)-1].t
	} else {
		if debug {
			fmt.Printf("---%+v\n", ev.mi)
		}
		var w *what
		w, ok := e.w[ev.mi.Moniker]
		if !ok {
			e.w[ev.mi.Moniker] = &what{uuid: ev.mi.Moniker}
			w = e.w[ev.mi.Moniker]
		}
		// adjust times
		if w.begin.IsZero() {
			w.begin = ev.mi.T
		}
		// do we have the history stuff?
		if ev.mi.Oflock != "" {
			xx := strings.Split(ev.mi.Oflock, " ")
			if len(xx) != 3 {
				fmt.Printf("bad format Oflock = %s\n", ev.mi.Oflock)
				return
			}
			w.n = xx[0]
			w.fname = xx[2]
		}
		e.e = append(e.e, ev)
		sort.Sort(e)
		// cutoff is the last timestamp minus the window size
		cutoff = e.e[len(e.e)-1].t.Add(-win)
	}
	var i int
	for i = range e.e {
		if e.e[i].t.After(cutoff) {
			break
		}
		// increment cursor if necessary
		for cursor.Before(e.e[i].mi.T) {
			// print before we bump
			print(cursor, cursorStep)
			cursor = cursor.Add(cursorStep)
		}
		//fmt.Printf("%d: %+v\n", i, e.e[i].mi)
		e.w[e.e[i].mi.Moniker].latest = e.e[i].mi.T
	}
	e.e = e.e[i:]
}

// sorting
type plist struct {
	w []*what
}

func (p *plist) Len() int {
	return len(p.w)
}
func (p *plist) Less(i, j int) bool {
	if p.w[i].n < p.w[j].n {
		return true
	}
	if p.w[i].n > p.w[j].n {
		return false
	}
	return p.w[i].fname < p.w[j].fname
}
func (p *plist) Swap(i, j int) {
	p.w[i], p.w[j] = p.w[j], p.w[i]
}

func print(c time.Time, step time.Duration) {
	fmt.Printf(">> cursor %s\n", c.String())
	var p plist
	for _, v := range e.w {
		p.w = append(p.w, v)
	}
	sort.Sort(&p)

	lnode := "_"
	for _, w := range p.w {
		if w.n == "" {
			continue
		}
		pn := ""
		if lnode != w.n {
			pn = w.n
			lnode = w.n
		}
		if w.latest.IsZero() {
			continue
		}
		warn := ""
		if w.latest.Sub(c) <= -2*step {
			warn = "   *******"
		}
		dif := float32(w.latest.Sub(c)) / float32(time.Second)
		fmt.Printf("%11s %15s (%13s): %s .. %7.3fs%s\n", pn, w.fname, w.uuid, w.begin.Format(time.StampMilli)[7:], dif, warn)
	}
}
