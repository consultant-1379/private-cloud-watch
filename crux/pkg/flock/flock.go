package flock

import (
	"encoding/hex"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/erixzone/crux/pkg/clog"
	"github.com/erixzone/crux/pkg/crux"
)

// we need random randomness
func init() {
	pid := os.Getpid()
	seed := int64(time.Now().Nanosecond()) * int64(pid)
	rand.Seed(seed)
	for i := time.Now().Second() % 1023; i > 0; i-- {
		rand.Int()
	}
}

var scruntLevel = 1

// shut up lint
const (
	voteMax      = 100000 // this is better if comfortably larger than the number of nodes
	rebootHearts = 4.0    // reboot time in heartbeat units
	histRework   = .1     // how much of your rework is history based
	Atime        = "04:05.000"
	shortLeader  = 2  // how many heartbeats to wait until the leader (who knows us) times out
	specLeader   = 10 // how many heartbeats to wait until the leader (who does not know us) times out
	stableLeader = 6  // a leader is stable after this many heartbeats

	VersionString = "2018-11-15 v.01"
)

// Flock structure (actually per member)
type Flock struct {
	fi            Info
	fn            Fnode
	ff            Fflock
	shutdown      chan bool // request shutdown
	finished      chan bool // actually shutdown
	rcv           chan *Info
	chkpt         checkpoint
	epochID       *Nonce // change this to trigger new session keys
	sec0, sec1    *Key   // shared secret for flock identification
	beacon        string
	mySteward     string
	myRegistry    string
	myRegistryKey string
	myYurt        string
	horde         string
	mems          map[string]mem
	visitor       bool
	sync.Mutex
}

// GetFflock provides access to internal Fflock field
func (f *Flock) GetFflock() Fflock {
	return f.ff
}

// GetCertificate : cert chain for TLS
func (f *Flock) GetCertificate() *crux.TLSCert {
	return f.fn.GetCertificate()
}

// Monitor gives access to the beacon
func (f *Flock) Monitor() chan crux.MonInfo {
	return f.fn.Monitor()
}

type checkpoint struct {
	sync.Mutex
	me   string   // my moniker
	mems []NodeID // my flock!
}

// Cmd is what op we pass into the flock.
type Cmd int

// oh shut the verifier up!
const (
	CmdReboot Cmd = iota
	CmdCandidate
)

// Flock1_0 is responsible for the side activities of networking. This includes the healthcheck activities
func Flock1_0(quit <-chan bool, alive chan<- []crux.Fservice, network **crux.Confab) {
	hbeat := time.NewTicker(3 * time.Second)
bigloop:
	for {
		select {
		case <-quit:
			(**network).Close()
			hbeat.Stop()
			break bigloop
		case <-hbeat.C:
			// send heartbeat on alive
		}
	}
	fmt.Printf("flock returning\n")
	return
}

// NewFlockNode is how we start up a flock node.
func NewFlockNode(me NodeID, fn Fnode, ff Fflock, sec0 *Key, beacon string, visitor bool) *Flock {
	f := Flock{fn: fn, ff: ff, sec0: sec0}
	f.fi.Me = me
	f.fn.SetMe(&f.fi.Me)
	f.rcv = make(chan *Info)
	f.shutdown = make(chan bool, 2) // serve + receiver
	f.finished = make(chan bool, cap(f.shutdown))
	f.chkpt = checkpoint{me: crux.SmallID(), mems: make([]NodeID, 0)}
	f.beacon = beacon
	f.mySteward = fmt.Sprintf("%s:%d", me.Moniker, 0)
	f.myRegistry = fmt.Sprintf("%s:%d", me.Moniker, 0)
	f.myRegistryKey, _ = Key2String(*sec0)
	f.mems = make(map[string]mem, 1)
	f.visitor = visitor
	f.Lock()
	f.Unlock()

	go f.serve()
	go f.receiver()
	fmt.Printf("returning newflocknode\n")
	return &f
}

// Close is how we shut down the node.
func (f *Flock) Close() {
	const n = 2 // serve + receiver
	// ask for shutdown
	for i := 0; i < n; i++ {
		f.shutdown <- true
	}
	// the receiver does a checkpoint as it shuts down
	// wait for shutdown
	for i := 0; i < n; i++ {
		<-f.finished
	}
}

// GetNames sets the node name
func (f *Flock) GetNames() crux.ConfabN {
	f.Lock()
	defer f.Unlock()
	return crux.ConfabN{
		Bloc:         f.fi.Flock,
		Horde:        f.horde,
		Node:         f.fi.Me.Moniker,
		Leader:       f.fi.Leader.Moniker,
		Stable:       f.fi.Lvote == voteMax,
		Steward:      f.mySteward,
		RegistryAddr: f.myRegistry,
		RegistryKey:  f.myRegistryKey,
		Yurt:         f.myYurt,
	}
}

// SetSteward sets the steward address
func (f *Flock) SetSteward(ip string, port int) {
	f.Lock()
	f.mySteward = fmt.Sprintf("%s:%d", ip, port)
	f.Unlock()
}

// SetRegistry set the netid/key for the registry
func (f *Flock) SetRegistry(ip string, port int, key string) {
	f.Lock()
	f.myRegistry = fmt.Sprintf("%s:%d", ip, port)
	f.myRegistryKey = key
	f.Unlock()
}

// SetYurt sets the gatewy port
func (f *Flock) SetYurt(ip string, port int) {
	f.Lock()
	f.myYurt = fmt.Sprintf("%s:%d", ip, port)
	f.Unlock()
}

// SetHorde sets the horde name
func (f *Flock) SetHorde(h string) {
	f.Lock()
	f.horde = h
	f.Unlock()
}

func (f *Flock) receiver() {
	for {
		if len(f.shutdown) > 0 { // something there?
			<-f.shutdown // make sure
			f.finished <- true
			return
		}
		info := f.fn.Recv()
		f.rcv <- info
	}
}

type mem struct {
	expire time.Time
	id     NodeID
}

// i apologise for this, but this routine implements the whole protocol.
// its looonnnggggg.
func (f *Flock) serve() {
	flock := &f.fi
	logk := clog.Log.With("focus", "flock_key", "node", flock.Me.Moniker)
	logg := clog.Log.With("focus", "flock_general", "node", flock.Me.Moniker)
	loge := clog.Log.With("focus", "flock_election", "node", flock.Me.Moniker)
	logl := clog.Log.With("focus", "flock_leader", "node", flock.Me.Moniker)
	var history map[string]NodeID
	var promotion time.Time // when i became leader
	monitor := f.fn.Monitor()
	var leaderExpire time.Time
	f.mems = make(map[string]mem)
	mycheckpoint := func(important bool) {
		if important {
			logg.Log(nil, "checkpointing %d members", len(f.mems))
			f.chkpt.Lock()
			f.chkpt.me = flock.Me.Moniker
			f.chkpt.mems = make([]NodeID, 0, len(f.mems))
			for _, x := range f.mems {
				f.chkpt.mems = append(f.chkpt.mems, x.id)
			}
			f.chkpt.Unlock()
			// go to persistent storage
		}
	}
	promote := func(leader bool) {
		if leader {
			promotion = time.Now().UTC()
			logl.Log("ptime", promotion, "flock", flock.Flock, "promote to leader")
		} else {
			var zero time.Time
			promotion = zero
		}
	}

	var tickKR *time.Timer
	var rotatePrimary bool
	timerKR := func() {
		if tickKR != nil && !tickKR.Stop() && len(tickKR.C) > 0 { // necessary?
			<-tickKR.C
		}
		tickKR = time.NewTimer(f.ff.KeyPeriod())
		rotatePrimary = false
	}
	setkeys := func() {
		f.fn.SetKeys(f.epochID, f.sec0, f.sec1)
		logk.Log("epochID", f.epochID, "sec0", f.sec0, "sec1", f.sec1, "node", f.fi.Dest.Moniker, "setkey")
		flock.EpochID = *f.epochID
		timerKR()
	}

	setSvcs := func() {
		f.Lock()
		f.fi.Steward = f.mySteward
		f.fi.Registry = f.myRegistry
		f.fi.RegistryKey = f.myRegistryKey
		f.Unlock()
		logl.Log(nil, "sending4 steward=%s registry=%s/%s", f.fi.Steward, f.fi.Registry, f.fi.RegistryKey)
	}
	startOver := func() {
		oflock := flock.Flock
		flock.Flock = "_" + crux.SmallID()
		if f.visitor {
			flock.Vote = 1
		} else {
			flock.Vote = 1 + rand.Int31n(voteMax-1)
		}
		flock.Me.Horde = ""
		flock.Leader = flock.Me
		flock.Lvote = flock.Vote
		flock.Expire = time.Now().UTC().Add(shortLeader * f.ff.Heartbeat())
		f.epochID, _ = NewGCMNonce()
		setkeys()
		leaderExpire = flock.Expire
		history = make(map[string]NodeID)
		mycheckpoint(true)
		logg.Log(nil, "restart v=%d flock=%s", flock.Vote, flock.Flock)
		if monitor != nil {
			mi := crux.MonInfo{Op: crux.LeaderStartOp, Moniker: flock.Me.Moniker, T: time.Now().UTC(), Flock: flock.Flock, Oflock: oflock, N: len(f.mems)}
			monitor <- mi
		}
		promote(true)
	}
	ping := func(dd NodeID) {
		flock.Dest = dd
		if flock.Dest.Moniker != flock.Me.Moniker {
			flock.Expire = time.Now().UTC().Add(shortLeader * f.ff.Heartbeat())
			f.fn.Send(flock)
			logk.Log(nil, "ping(%s) using %s", flock.Dest.Moniker, f.epochID)
		}
	}
	// now a bevy of other timer functions
	var tickHB, tickCPT, tickMW, tickPR *time.Timer
	timerHB := func() { tickHB = time.NewTimer(f.ff.Heartbeat()) }
	timerMW := func() { tickMW = time.NewTimer(f.ff.Probebeat()) }
	timerCPT := func() { tickCPT = time.NewTimer(f.ff.Checkpoint()) }
	timerPR := func() { tickPR = time.NewTimer(f.ff.HistoryPrune()) }
	timerStop := func() { tickHB.Stop(); tickCPT.Stop(); tickPR.Stop(); tickMW.Stop() }

	startOver()
	mycheckpoint(true)
	timerHB()
	timerMW()
	timerCPT()
	timerPR()
	timerKR()

forloop:
	for {
		select {
		case <-tickKR.C:
			logg.Log(nil, "key timer")
			timerKR()
			rotatePrimary = true
		case <-tickHB.C:
			timerHB()
			if (monitor != nil) && (len(f.mems) > 0) {
				mi := crux.MonInfo{Op: crux.LeaderHeartOp, Moniker: flock.Me.Moniker, T: time.Now().UTC(), Flock: flock.Flock, N: len(f.mems)}
				monitor <- mi
			}
			t := time.Now().UTC()
			logg.Log(nil, "heartbeat timer %s", t.Format(Atime))
			if leaderExpire.Before(t) && (flock.Me.Moniker != flock.Leader.Moniker) {
				loge.Log("leader", flock.Leader.Moniker, "flock", flock.Flock, "leaderExpire", leaderExpire, "leader expired")
				old := flock.Leader
				startOver()
				// remind old leader
				ping(old)
				continue
			}
			flock.Expire = t.Add(shortLeader * f.ff.Heartbeat())
			if flock.Leader.Moniker != flock.Me.Moniker { // am i the leader?
				logg.Log("leader", flock.Leader.Moniker, "heartbeat to leader")
				setSvcs()
				ping(flock.Leader)
				promote(false)
				continue
			}
			// we are leader, hear me roar!
			// first, lets deal with leadership stability
			if promotion.IsZero() {
				// we weren't leader last heartbeat, so set promotion time
				promote(true)
			} else {
				// have we been leader for long enough
				celebration := promotion.Add(stableLeader * f.ff.Heartbeat())
				if time.Now().UTC().After(celebration) && (flock.Vote != voteMax) {
					flock.Vote = voteMax
					flock.Lvote = voteMax
					logl.Log("ctime", celebration, "flock", flock.Flock, nil, "promote vote to max %d", voteMax)
				}
			}
			setSvcs() // reset my services
			if len(f.mems) > 0 {
				// get on with heartbeating
				logl.Log("flock", flock.Flock, nil, "++++ heartbeat to %d members", len(f.mems))
				leaderExpire = flock.Expire.Add(shortLeader * f.ff.Heartbeat())
				if rotatePrimary {
					if x, err := NewGCMNonce(); err == nil {
						f.epochID = x
						flock.EpochID = *f.epochID
						logk.Log("epochID", f.epochID.String(), "rotatekey")
						setkeys()
					}
				}
				// update all my members
				for _, n := range f.mems {
					ping(n.id)
				}
			}
		case <-tickMW.C:
			logg.Log("makework")
			np := 0 // number of pings
			n := f.ff.ProbeN()
			nh := int(float32(n) * histRework)
			// deal with edge conditions
			if nh < 1 {
				nh = 1
			}
			if n <= 1 {
				n = 1
				nh = 0
			}
			slots := make([]string, n)
			// look for history addresses not in current member list
			// pick nh addresses via reservoir sampling
			nvalid := 0
			for _, h := range history {
				if _, ok := f.mems[h.Moniker]; ok {
					continue
				}
				if h.Moniker == flock.Leader.Moniker {
					continue
				}
				nvalid++
				if np < nh {
					slots[np] = h.Addr
					np++
				} else {
					x := rand.Intn(nvalid)
					if x < nh {
						slots[x] = h.Addr
					}
				}
			}
			// generate a dont use list
			nup := make(map[string]bool)
			for _, h := range history {
				nup[h.Addr] = true
			}
			for _, h := range f.mems {
				nup[h.id.Addr] = true
			}
			// now just fill up with random probes
			ntries := 5 * n // but only this number of tries
			for (np < n) && (ntries > 0) {
				ntries--
				x, ok := f.ff.Probe()
				if ok {
					slots[np] = x
					np++
				}
			}
			if monitor != nil {
				mi := crux.MonInfo{Op: crux.ProbeOp, Moniker: flock.Me.Moniker, T: time.Now().UTC(), Flock: flock.Flock, N: np}
				monitor <- mi
			}
			// now send them
			var info NodeID
			for i := 0; i < np; i++ {
				info.Addr = slots[i]
				info.Moniker = info.Addr
				ping(info)
			}
			logg.Log(nil, "sent %d pings", np)
			timerMW()
		case <-tickCPT.C:
			mycheckpoint(true)
			timerCPT()
		case <-f.shutdown:
			break forloop
		case <-tickPR.C:
			if len(f.mems) > 0 {
				now := time.Now().UTC()
				n := 0
				for x, k := range f.mems {
					if k.expire.Before(now) {
						delete(f.mems, x)
						n++
						logl.Log("flock", flock.Flock, nil, "pruning expired %s", k.id.Moniker)
					}
				}
				if (n > 0) && (monitor != nil) {
					mi := crux.MonInfo{Op: crux.LeaderDeltaOp, Moniker: flock.Me.Moniker, T: time.Now().UTC(), Flock: flock.Flock, N: len(f.mems)}
					monitor <- mi
				}
				logl.Log("flock", flock.Flock, nil, "pruned %d expired mems", n)
				mycheckpoint(true)
			}
			timerPR()
		case probe := <-f.rcv:
			if probe.Vote == 0 {
				// forgive this hack
				logg.Log("probe", *probe, "admin")
				switch Cmd(probe.Lvote) {
				case CmdReboot:
					if monitor != nil {
						mi := crux.MonInfo{Op: crux.RebootOp, Moniker: flock.Me.Moniker, T: time.Now().UTC(), Flock: flock.Flock}
						monitor <- mi
					}
					rtime := time.Duration((0.8 + .4*rand.Float64()) * rebootHearts * float64(f.ff.Heartbeat()))
					logg.Log(nil, "rebooting! duration=%s", rtime.String())
					time.Sleep(rtime) // it takes time to reboot
					startOver()
					continue forloop
				case CmdCandidate:
					// just send ourself to the address
					flock.Expire = time.Now().UTC().Add(specLeader * f.ff.Heartbeat())
					ping(probe.Leader)
				}
				continue
			}
			history[probe.Me.Moniker] = probe.Me
			if probe.Me.Moniker == flock.Me.Moniker {
				// log me? TBD
				continue
			}
			t := time.Now().UTC()
			if probe.Expire.Before(t) {
				logg.Log(nil, "probe expired: %+v prior to %+v", probe.Expire, t)
				continue
			}
			logg.Log("node", flock.Me.Moniker, "focus", "flock_probe", nil, "probe(%s:%s ldr=%s lv=%d v=%d exp=%s) me(%s ldr=%s lv=%d v=%d ldrexp=%s)",
				probe.Flock, probe.Me.Moniker, probe.Leader.Moniker, probe.Lvote, probe.Vote, probe.Expire.Format(Atime),
				flock.Flock, flock.Leader.Moniker, flock.Lvote, flock.Vote, leaderExpire.Format(Atime))
			// honor any horde changes
			if probe.Dest.Horde != "" {
				logg.Log(nil, "setting %s.Horde to %s", probe.Me.Moniker, probe.Dest.Horde)
				f.horde = probe.Dest.Horde
			}
			flock.Expire = t.Add(f.ff.Heartbeat()) // we'll be sending me out; set expire
			if probe.Flock > flock.Flock {
				old := flock.Leader
				loge.Log("nflock", probe.Flock, "nleader", probe.Leader.Moniker, "flock", flock.Flock, "join new flock")
				if monitor != nil {
					mi := crux.MonInfo{Op: crux.JoinOp, Moniker: flock.Me.Moniker, T: time.Now().UTC(), Oflock: flock.Flock, Flock: probe.Flock, N: len(f.mems)}
					monitor <- mi
				}
				flock.Flock = probe.Flock
				// if we're joining a new flock and we had promoted, pick a new vote
				if flock.Vote == voteMax {
					if f.visitor {
						flock.Vote = 1
					} else {
						flock.Vote = 1 + rand.Int31n(voteMax-1)
					}
					loge.Log("vote", flock.Vote, "set new vote (because we had promoted)")
				}
				if (probe.Lvote >= flock.Vote) || (flock.Vote == voteMax) {
					// pick up probe's leader
					flock.Leader = probe.Leader
					flock.Lvote = probe.Lvote
					flock.Steward = probe.Steward
					flock.Registry = probe.Registry
					flock.RegistryKey = probe.RegistryKey
					logl.Log(nil, "inheriting leader=%s lvote=%d steward=%s registry=%s/%s", flock.Leader, flock.Lvote,
						flock.Steward, flock.Registry, flock.RegistryKey)
					promote(false)
				} else {
					// apparently we are leader
					flock.Leader = flock.Me
					flock.Lvote = flock.Vote
					promote(true)
				}
				// new epoch
				f.epochID = new(Nonce)
				*(f.epochID) = probe.EpochID
				setkeys()
				// let all my mems (if any) know
				for _, k := range f.mems {
					ping(k.id)
				}
				logl.Log("flock", flock.Flock, "clearing mems")
				f.mems = make(map[string]mem)
				mycheckpoint(false)
				// let new leader know
				ping(probe.Leader)
				// let old leader know as well
				ping(old)
				leaderExpire = time.Now().UTC().Add(shortLeader * f.ff.Heartbeat())
				continue
			}
			if probe.Flock < flock.Flock {
				loge.Log("nnode", probe.Me.Moniker, "nflock", probe.Flock, "flock", flock.Flock, "ping flock to join us")
				// send us back to the probe
				ping(probe.Me)
				continue
			}
			loge.Log(nil, "vote %d <> lvote %d", probe.Vote, flock.Lvote)
			// first, check for a weird case: two leaders in the same flock!
			// this can happen when a flock starts accumulating around two different high-scoring members.
			// you can try complicated things, but starting over is quick, and they'll rejoin soon
			// because of all teh nodes that know about them.
			if (probe.Lvote == voteMax) && (flock.Lvote == voteMax) && (probe.Leader.Moniker != flock.Leader.Moniker) {
				loge.Log(nil, "two leaders in flock; we're going to startover")
				startOver()
				continue
			}
			// in our group; new leader?
			if probe.Vote > flock.Lvote {
				loge.Log("nleader", probe.Me.Moniker, "flock", flock.Flock, nil, "new leader (%d > %d)", probe.Vote, flock.Lvote)
				flock.Leader = probe.Me
				flock.Lvote = probe.Vote
				flock.Steward = probe.Steward
				flock.Registry = probe.Registry
				flock.RegistryKey = probe.RegistryKey
				// let probe know
				ping(probe.Me)
				leaderExpire = flock.Expire.Add(specLeader * f.ff.Heartbeat()) // extra time until we hear from her
				// let all my mems (if any) know
				for _, k := range f.mems {
					ping(k.id)
				}
				f.mems = make(map[string]mem)
				mycheckpoint(false)
				continue
			}
			// is it our leader checking in?
			if probe.Me.Moniker == flock.Leader.Moniker {
				logg.Log("leader checking in")
				leaderExpire = flock.Expire.Add(shortLeader * f.ff.Heartbeat())
				var changed bool
				if probe.EpochID != *f.epochID {
					f.epochID = new(Nonce)
					*(f.epochID) = probe.EpochID
					changed = true
				}
				if probe.Beacon != f.beacon {
					f.beacon = probe.Beacon
					changed = true
				}
				if changed {
					setkeys()
				}
				flock.Steward = probe.Steward
				flock.Registry = probe.Registry
				flock.RegistryKey = probe.RegistryKey
				flock.Lvote = probe.Lvote
				logl.Log(nil, "heartbeat: leader=%s lvote=%d steward=%s registry=%s", flock.Leader, flock.Lvote, flock.Steward, flock.Registry)
				f.Lock()
				f.myRegistry = flock.Registry
				f.myRegistryKey = flock.RegistryKey
				f.mySteward = flock.Steward
				f.Unlock()
			}
			if (probe.Leader.Moniker == flock.Leader.Moniker) && (flock.Me.Moniker == flock.Leader.Moniker) {
				logl.Log("mem", probe.Me.Moniker, "flock", flock.Flock, "adding mem")
				f.mems[probe.Me.Moniker] = mem{expire: probe.Expire, id: probe.Me}
				mycheckpoint(false)
				continue
			}
			if probe.Lvote > flock.Lvote {
				loge.Log("nleader", probe.Leader.Moniker, "flock", flock.Flock, "new leader(%s.%d > %s.%d)",
					probe.Leader.Moniker, probe.Lvote, flock.Leader.Moniker, flock.Lvote)
				// change the leader
				flock.Leader = probe.Leader
				flock.Lvote = probe.Lvote
				flock.Steward = probe.Steward
				flock.Registry = probe.Registry
				flock.RegistryKey = probe.RegistryKey
				// tell probe's leader
				ping(probe.Leader)
				leaderExpire = flock.Expire.Add(specLeader * f.ff.Heartbeat()) // extra time until we hear from her
				// let all my mems (if any) know
				for _, k := range f.mems {
					ping(k.id)
				}
				// reset our membership list
				f.mems = make(map[string]mem)
				mycheckpoint(false)
				continue
			}
			if probe.Lvote < flock.Lvote {
				loge.Log("flock", flock.Flock, "probe has bad leader (%d < %d); ping back", probe.Lvote, flock.Lvote)
				ping(probe.Me)
			}
		}
	}
	timerStop()
	mycheckpoint(true)
	logg.Log("quitting")
	f.finished <- true
}

// Mem is how we look at the internal mems structure.
func (f *Flock) Mem() []NodeID {
	f.Lock()
	defer f.Unlock()
	clog.Log.Log(nil, "flock.mems = %v", f.mems)
	mems := make([]NodeID, 0, len(f.mems))
	for _, x := range f.mems {
		mems = append(mems, x.id)
	}
	return mems
}

// Reboot is how we reboot the flock.
func (f *Flock) Reboot() {
	x := f.fi
	x.Dest = x.Me
	x.Vote = 0
	x.Lvote = int32(CmdReboot)
	f.fn.Send(&x)
	clog.Log.Log("node", x.Dest.Moniker, "reboot")
}

func (k *Key) String() string {
	if k == nil {
		return "<nil>"
	}
	x := *k
	return hex.EncodeToString([]byte(x[0:8]))[0:8]
}

// String2Key returns a Key from a string. surprisingly tricky
func String2Key(s string) (Key, *crux.Err) {
	var key Key
	bits, err := hex.DecodeString(s)
	if err != nil {
		return key, crux.ErrE(err)
	}
	n := len(bits)
	if n > len(key) {
		return key, crux.ErrF("string for Key too large")
	}
	// if n != 32, then we have a leading zeros issue
	copy(key[len(key)-n:], bits)
	return key, nil
}

// Key2String  returns a string from a Key type
func Key2String(k Key) (string, *crux.Err) {
	str := hex.EncodeToString(k[:32])
	if strings.Count(str, "0") == 64 {
		return "", crux.ErrF("parameter error - empty key")
	}
	return str, nil
}

// String from Nonce
func (n *Nonce) String() string {
	return fmt.Sprintf("%x", *n)
}
