package khan

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/horde"
	"github.com/erixzone/crux/pkg/kv"
)

const (
	keySpec    = "khan/spec"    // target service specification
	keyConform = "khan/conform" // does actual conform to target; either “ok” or an explanation
	keyProbe   = "khan/crux"    // time of last evaluation of services
	keyPeriod  = "khan/period"  // time between evaluation of services
	keyHent    = "khan/hent"    // maximum number of entries in the history
	keyHistory = "khan/history" // the last several actions (with timestamps) by khan
)

// Khan looks up the spec of what it is to do from the KV store
// and performs those actions via the Action interface.
// it maintains various variables under /khan.
// it returns an indication of whether anything was done.
// an error is returned if anything went awry EXCEPT for a missing spec;
// in this case, it returns no error and no activity.
func Khan(adm horde.Administer, kv kv.KV, act horde.Action, pending []horde.Service) (active bool, who []string, err *crux.Err) {
	spec, e := kv.Get(keySpec)
	if spec == "" {
		// nothing to do
		return
	}
	if e != nil {
		err = e
		return
	}
	cruxT := time.Now()
	diff, _, w3, explain, err := diff(adm, act, spec, pending)
	if err != nil {
		fmt.Printf("Khan(>>%s<<) returns %s\n", spec, err.String())
		return
	}
	fmt.Printf("Khan-explain: %s\n", explain)
	// build list of who will be started
	wmap := make(map[string]int, 0)
	for _, w := range w3 {
		wmap[w.service] += w.count
	}
	for k := range wmap {
		who = append(who, k)
	}
	/* change this to logging
	for _, w := range diff {
		 fmt.Printf("KHAN diff: %s\n", w.String())
	}
	*/
	hist, ee := kv.Get(keyHistory)
	if ee != nil {
		hist = ""
	}
	history := strings.Split(hist, "\n")
	var nStart, nStop int
	act.Reset()
	fmt.Printf("diff-enact: %+v\n", diff)
	for _, w := range diff {
		var did string
		if w.count > 0 {
			act.Start(w.node, w.service, w.count)
			did = fmt.Sprintf("%s start %s %s %d", time.Now().String(), w.node, w.service, w.count)
			nStart += w.count
		} else {
			act.Stop(w.node, w.service, -w.count)
			did = fmt.Sprintf("%s stop %s %s %d", time.Now().String(), w.node, w.service, -w.count)
			nStop += -w.count
		}
		history = append(history, did)
	}
	var conform string
	active = (nStart + nStop) > 0
	if active {
		conform = fmt.Sprintf("need to start %d and stop %d instances", nStart, nStop)
	} else {
		conform = "ok"
	}
	ee = kv.Put(keyConform, conform)
	if ee != nil {
		err = e
		return
	}
	hh, ee := kv.Get(keyHent)
	if ee != nil {
		hh = ""
	}
	hent := 10000 // default if no good number
	x, ef := strconv.Atoi(hh)
	if ef == nil {
		hent = x
	} else {
		// log an error
	}
	if hent < len(history) {
		history = history[len(history)-hent:]
	}
	ee = kv.Put(keyHistory, strings.Join(history, "\n"))
	if ee != nil {
		err = ee
		return
	}
	ee = kv.Put(keyProbe, cruxT.String())
	if ee != nil {
		err = ee
	}
	return
}

// Period returns the periodicity for enforcing the Khan cruxs.
// as per the documentation, it is interpreted as the number of seconds.
func Period(kv kv.KV) (int, *crux.Err) {
	hh, err := kv.Get(keyPeriod)
	if err != nil {
		return 0, err
	}
	x, e := strconv.Atoi(hh)
	if e != nil {
		return 0, crux.ErrE(e)
	}
	return x, nil
}
