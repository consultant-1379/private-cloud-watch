package lib

import (
	"fmt"
	"io/ioutil"
	"strings"
	"sync"
)

const bboxSep = "|" // separate fields in blackbox items

// BlackBox is sort of a generic (flight) recorder for all sorts of events
type BlackBox interface {
	Record(string, interface{})
	Playback() []string
	MemToFielded([]string) [][]string
	FileToFielded(string) ([][]string, error)
	ToFile(string, []string) error
}

type item struct {
	lab string
	v   interface{}
}

// Deck is our implementation of a BlackBox
type Deck struct {
	sync.Mutex
	items []item
}

// NewBlackBox returns a Deck
func NewBlackBox() *Deck {
	var d Deck
	d.items = make([]item, 0, 2)
	return &d
}

// Record saves any sort of event
func (d *Deck) Record(lab string, x interface{}) {
	d.Lock()
	d.items = append(d.items, item{lab: lab, v: x})
	d.Unlock()
}

// Playback returns string versions of saved events
func (d *Deck) Playback() []string {
	var s []string
	d.Lock()
	for _, i := range d.items {
		s = append(s, i.String())
	}
	d.Unlock()
	return s
}

// String (unfortunately) has to know about the types of events
func (i item) String() string {
	s := i.lab + bboxSep
	switch t := i.v.(type) {
	case Chore:
		s += fmt.Sprintf("exec")
		for _, e := range t.InEnts {
			s += fmt.Sprintf("%s%s%s%s", bboxSep, e.Name, bboxSep, e.Hash.String())
		}
	case EventStatus:
		s += fmt.Sprintf("status%s%s%s%s", bboxSep, t.Status.String(), bboxSep, t.Err)
	case EventControl:
		s += fmt.Sprintf("control%s%s", bboxSep, t.Op.String())
	case EventFS:
		switch t.Op {
		case FSEnormal:
			s += fmt.Sprintf("fs%s+%s%s%s%s", bboxSep, bboxSep, t.Path, bboxSep, t.Hash.String())
		case FSEdelete:
			s += fmt.Sprintf("fs%s-%s%s%s%s", bboxSep, bboxSep, t.Path, bboxSep, t.Hash.String())
		case FSEexecstatus:
			s += fmt.Sprintf("execret%s%s", bboxSep, t.Err)
		}
	default:
		s += fmt.Sprintf("unknown type!")
	}
	return s
}

// MemToFielded converts a string slice (as gen by Playback) into a fielded format for analysis
func (d *Deck) MemToFielded(m []string) [][]string {
	var ret [][]string
	for _, x := range m {
		ret = append(ret, strings.Split(x, bboxSep))
	}
	return ret
}

// FileToFielded generates a fielded format from a file
func (d *Deck) FileToFielded(path string) ([][]string, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var ret [][]string
	for _, x := range strings.Split(string(b), "\n") {
		fields := strings.Split(x, bboxSep)
		if len(fields) > 1 {
			ret = append(ret, fields)
		}
	}
	return ret, nil
}

// ToFile saves a Playback output to a file.
func (d *Deck) ToFile(path string, m []string) error {
	// i believe in ioutil
	var out string
	for _, x := range m {
		out += x + "\n"
	}
	return ioutil.WriteFile(path, []byte(out), 0664)
}
