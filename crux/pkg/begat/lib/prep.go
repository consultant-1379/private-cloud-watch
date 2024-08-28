package lib

import (
	"fmt"
	"github.com/erixzone/crux/pkg/crux"
)

/*
	this takes a Parse, which is a compiled form of the original begatfile,
and returns the corresponding set of chores (in a topological sorted order).

	the parse has already been linearised (in parse.code)
and so all we need do is handle the ordering implied by inputs and outputs.
*/

func (p *Parse) prep(targets []string) ([][]*Chore, error) {
	// set up a more efficient way to see if a string is a target
	target := make(map[string]bool, len(targets))
	for _, t := range targets {
		target[t] = true
	}
	// set up a set of Chores so that we know what we're dealing with
	work := make(map[string]*Chore)
	for _, s := range p.code {
		fmt.Printf("=> %s %+v\n", s.What, s)
		crux.Assert(s.What == StatementCallDict)
		c := Chore{
			RunID:     crux.LargeID(),
			D:         s.Dict,
			Dir:       s.Dir,
			Cacheable: true,
		}
		m, err := makeMounts(s.Mount)
		if err != nil {
			return nil, err
		}
		c.Mounts = m
		// see if we're a target (and therefore must make)
		for _, o := range c.D.Outputs {
			if target[o] {
				c.Cacheable = false
				break
			}
		}
		work[c.RunID] = &c
	}
	// we're nearly there. we're just a topological sort away ...
	// build a map from input to []*Chore (there may be multiple!)
	tsort := NewTsort()
	imap := make(map[string][]*Chore, 2*len(work))
	for _, w := range work {
		for _, i := range w.D.Inputs {
			x, ok := imap[i]
			if ok {
				x = append(x, w)
			} else {
				x = make([]*Chore, 1)
				x[0] = w
			}
			imap[i] = x
		}
	}
	// now we just run through work again, this time looking at outputs
	for _, w := range work {
		for _, o := range w.D.Outputs {
			if x, ok := imap[o]; ok {
				for _, y := range x {
					y.D.Depends = append(y.D.Depends, o)
					tsort.Pair(y.RunID, w.RunID)
				}
			}
		}
	}
	// just about done
	sorted := tsort.Order()
	ret := make([][]*Chore, len(sorted))
	for i, list := range sorted {
		ret[i] = make([]*Chore, len(list))
		for j := range list {
			ret[i][j] = work[list[j]]
		}
	}
	// ich bin outta here
	return ret, nil
}

func makeMounts(ms []*Statement) ([]Mount, error) {
	m := make([]Mount, len(ms))
	for i := range ms {
		if len(ms[i].Args) < 2 {
			return nil, fmt.Errorf("insufficent args in mount")
		}
		m[i] = Mount{Where: ms[i].Args[0], What: ms[i].Args[1]}
		if len(ms[i].Args) > 2 {
			m[i].Args = ms[i].Args[2:]
		}
	}
	return m, nil
}
