/*
	this implemntation does enough to run test4.begat.
	there are numerous TBDs, especially around looking up historical
	records. we'll get that next time round.
	also, the conversion of Printfs to logging will happen next time
	(when the logging stuff is stable.)
*/
/*
	the governing documentation for this is in doc.go

The pretendable flag has to be passed in because its value depends on the context in which begat is running.
For example, generally, it should be true, but if begat is invoked to produce a specific file, and that file
is an output file of this chore, then it is false.
*/

package lib

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/erixzone/crux/pkg/clog"
)

// stupid lint
const (
	MaxExec = 5 // limit on cyclic executions
)

func (ch *Chore) execute(pretendable bool, status chan<- EventStatus, ctl chan EventControl, log clog.Logger, fsi BIfs, hist BIhistory, exec BIexec, bbox *Bbox) {
	//clog := log.Log("who", ch.RunID)
	clog := log
	clog.Logi(nil, "starting: dir=%s pretendable=%v (output=%+v)", ch.Dir, pretendable, ch.D.Outputs)
	ch.Nexec = 0
	fsmap := make(map[string]*Ent) // our files
	fs := make(chan EventFS, 99)
	ofs := make(chan EventFS, 99)
	fsi.Pub("chore_"+ch.RunID, ofs)
	isInput := make(map[string]bool)
	// preload with environment "defaults"
	go exec.PrimeFS(ch)
	// register for updates
	var paths []string
	ch.InEnts = make([]Ent, len(ch.D.Inputs))
	for i, p := range ch.D.Inputs {
		path := filepath.Join(ch.Dir, p)
		isInput[path] = true
		ch.InEnts[i] = Ent{Name: path, Depend: mem(p, ch.D.Depends)}
		paths = append(paths, path)
	}
	ch.OutEnts = make([]Ent, len(ch.D.Outputs))
	for i, p := range ch.D.Outputs {
		path := filepath.Join(ch.Dir, p)
		ch.OutEnts[i] = Ent{Name: path}
		paths = append(paths, path)
	}
	fsi.Sub(FSRouterCmd{Op: FSRopen, ID: ch.RunID, Dest: fs, Files: paths})
	ch.RunEnts = make([]Ent, len(ch.InEnts)) // we'll populate later
	execOkay := true
	ch.setStatus(status, StatusStart, "", bbox)
	/*
		this is straightforward. we sit and absorb filesystem events while waiting
		for a prod to crank the classification engine.
	*/
bigloop:
	for {
		// we only want to wait for the very first probe and then process all we can
	eloop:
		for first := true; first || (len(fs)+len(ctl) > 0); first = false {
			select {
			case ce := <-ctl:
				clog.Logf(nil, "+++ ctlop: %+v", ce)
				if bbox.tape != nil && (ce.Op != OpCrank) { // check this
					bbox.tape.Record(bbox.label, ce)
				}
				switch ce.Op {
				case OpQuit:
					ctl = ce.Return
					break bigloop
				case OpCrank:
					prog, need := ch.step(status, fsmap, execOkay, ofs, exec, hist, bbox)
					//if ch.Status != StatusRunning {
					if ce.Return != nil {
						ce.Return <- EventControl{Op: OpCrank, Progress: prog, MustBuild: need}
					}
				case OpExec:
					execOkay = ce.Progress
					ce.Return <- EventControl{Op: OpExec}
					clog.Logf(nil, "set execOkay to %v", execOkay)
				default:
					ch.setStatus(status, StatusError, fmt.Sprintf("unknown ctl event: %d", ce.Op), bbox)
					continue eloop
				}
			case fe := <-fs:
				if bbox.tape != nil {
					bbox.tape.Record(bbox.label, fe)
				}
				if fe.Op == FSEexecstatus {
					clog.Logf(nil, "run status %+v\n", fe)
					if fe.Err != "" {
						ch.setStatus(status, StatusError, fe.Err, bbox)
						continue eloop
					}
					if ch.Status != StatusRunning {
						ch.setStatus(status, StatusError, "can't happen: runevent && !running", bbox)
						continue eloop
					}
					ch.setStatus(status, StatusExecuted, "", bbox)
					ctl <- EventControl{Op: OpCrank, Progress: true, MustBuild: false}
					t := Travail{Chore: *ch}
					hist.PutTravail(&t)
				} else {
					fmt.Printf("fs %+v (dir=%s)\n", fe, ch.Dir)
					ch.absorb(fe, fsmap)
				}
			}
		}
	}
	// we're done, shut down
	ch.setStatus(status, StatusQuit, "", bbox)
	ctl <- EventControl{Op: OpQuit}
}

func (ch *Chore) setStatus(status chan<- EventStatus, s StatusType, err string, bbox *Bbox) {
	ch.Status = s
	es := EventStatus{T: time.Now().UTC(), ID: ch.RunID, Status: ch.Status, Err: err}
	status <- es
	if bbox.tape != nil {
		bbox.tape.Record(bbox.label, es)
	}
	fmt.Printf("pubstatus %+v\n", es)
}

func (ch *Chore) step(status chan<- EventStatus, db map[string]*Ent, execOkay bool, fs chan EventFS, exec BIexec, hist BIhistory, bbox *Bbox) (progress bool, need bool) {
	progress = false
	need = false
	// check to see if we have a governing travail
	t, err := hist.GetTravail(ch)
	if err != nil {
		// log error
		t = nil
	}
	// check to see if we can no longer pretend
	mustbuild := false
	for _, o := range ch.OutEnts {
		mustbuild = mustbuild || (o.Status == EntNeed)
	}
	if (ch.Status == StatusPretend) && (mustbuild || (t == nil)) {
		ch.setStatus(status, StatusWaiting, "", bbox)
		changed := false
		// change all PRETEND outputs to MISSING
		for _, o := range ch.OutEnts {
			if o.Status == EntPretend {
				o.Status = EntMissing
				changed = true
			}
		}
		// change all pretend inputs to MUSTBUILD or MISSING
		for _, i := range ch.InEnts {
			if i.Status == EntPretend {
				if mustbuild {
					i.Status = EntNeed
					// must propagate as well
					// TBD
					need = true
				} else {
					i.Status = EntMissing
				}
				changed = true
			}
		}
		if changed {
			progress = true
			t = nil
		}
		// fall through and see if we can progress any other way
	}
	// can we pretend? note that we can go from PRETEND to WAITING and back to PRETEND
	if (t != nil) && ((ch.Status == StatusStart) || (ch.Status == StatusPretend) || (ch.Status == StatusWaiting)) {
		ch.setStatus(status, StatusPretend, "", bbox)
		ch.setPretend(db, t, fs)
		progress = true
		// becoming a PRETEND is enough, so return
		return
	}
	// are we ready to run?
	allthere := true // because they are if there aren't any
	for _, i := range ch.InEnts {
		allthere = allthere && (i.Status == EntExist)
	}
	if ((ch.Status == StatusStart) || (ch.Status == StatusWaiting)) && allthere {
		ch.setStatus(status, StatusCanRun, "", bbox)
		progress = true
	}
	// have we executed but some inputs have changed
	fschanged := false
	for i := range ch.InEnts {
		fschanged = fschanged || (ch.InEnts[i] != ch.RunEnts[i])
	}
	if (ch.Status == StatusExecuted) && fschanged {
		ch.setStatus(status, StatusCanRun, "", bbox)
		progress = true
	}
	// finally, can we actually run??
	if (ch.Status == StatusCanRun) && execOkay {
		if ch.Nexec >= MaxExec {
			// error TBD
		}
		ch.setStatus(status, StatusRunning, "", bbox)
		ch.RunEnts = make([]Ent, len(ch.InEnts))
		copy(ch.RunEnts[:], ch.InEnts[:])
		if bbox != nil {
			bbox.tape.Record(bbox.label, *ch)
		}
		fmt.Printf("about to exec %+v\n", *ch)
		if err := exec.Exec(ch); err != nil {
			// TBD: do something here
		}
	}
	return
}

func (ch *Chore) setPretend(db map[string]*Ent, t *Travail, fs chan EventFS) {
	//set all outputs in [MISSING,PRETEND] to be PRETEND with corresponding checksums from t
	// first, populate our notion of teh filesystem with the travail's
	for _, e := range t.OutEnts {
		x, ok := db[e.Name]
		if ok && !((x.Status == EntPretend) || (x.Status == EntMissing)) {
			continue
		}
		db[e.Name] = &Ent{Status: EntPretend, Name: e.Name, Hash: e.Hash}
		fs <- EventFS{Op: FSEnormal, Path: e.Name, Hash: e.Hash, Err: ""}
	}
	// now populate the OutEnts
	for i, e := range ch.D.OutEnts {
		x, ok := db[e.Name]
		if ok && !((x.Status == EntPretend) || (x.Status == EntMissing)) {
			continue
		}
		if ok {
			ch.D.OutEnts[i] = x
		}
	}
}

func (ch *Chore) absorb(fe EventFS, fsmap map[string]*Ent) {
	fmt.Printf("%s: fs %+v (dir=%s)\n", ch.RunID, fe, ch.Dir)
	name := fe.Path
	f, ok := fsmap[name]
	if ok && (f.Hash == fe.Hash) {
		return
	}
	fmt.Printf("redo: ok=%v path=%s hash: new=%d\n", ok, name, fe.Hash)
	e := Ent{Status: EntExist, Name: name, Hash: fe.Hash}
	switch fe.Op {
	case FSEnormal:
		e.Status = EntExist
	case FSEdelete:
		e.Status = EntMissing
	}
	fsmap[name] = &e
	ch.update(&e)
}

func (ch *Chore) update(fs *Ent) {
	for i, e := range ch.InEnts {
		if e.Name == fs.Name {
			fs.Depend = e.Depend
			fmt.Printf("ent update: %s: %+v\n", ch.RunID, *fs)
			ch.InEnts[i] = *fs
		}
	}
	for i, e := range ch.OutEnts {
		if e.Name == fs.Name {
			fmt.Printf("ent update: %s: %+v\n", ch.RunID, *fs)
			ch.OutEnts[i] = *fs
		}
	}
}
