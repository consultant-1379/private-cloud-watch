package flock

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/erixzone/crux/pkg/clog"
	"github.com/erixzone/crux/pkg/crux"
)

// StatusAnalyser returns an analysis of MonInfo's.
// CWVH period is not used. stable could use a line or two of documentation
func StatusAnalyser(quit chan bool, mon chan crux.MonInfo, stat chan Status, period int, stable time.Duration) {
	var stableT = time.Duration(stable)
	world := make(map[string]crux.MonInfo)
	reboot := make(map[string]bool)
	var genghis string
	var tstable, tstart time.Time
	log := clog.Log.Log("where", "analysis")

loop:
	for {
		select {
		case what := <-quit:
			if what {
				break loop
			}
			// reset stuff
			world = make(map[string]crux.MonInfo)
			//tstable = mi.T.Add(100 * time.Minute) // prevent premature misfire
		case mi := <-mon:
			switch mi.Op {
			case crux.JoinOp:
				log.Log("node", mi.Moniker, "flock", mi.Flock, "oflock", mi.Oflock, "change flock")
				//fmt.Printf("%s %s: %s -> %s\n", mi.T.Format(flock.Atime), mi.Moniker, mi.Oflock, mi.Flock)
			case crux.LeaderStartOp:
				world[mi.Flock] = mi
				delete(reboot, mi.Moniker)
				log.Log("node", mi.Moniker, "flock", mi.Flock, "oflock", mi.Oflock, "start new flock")
				//fmt.Printf("%s %s: start %s (old=%s)\n", mi.T.Format(flock.Atime), mi.Moniker, mi.Flock, mi.Oflock)
			case crux.LeaderHeartOp, crux.LeaderDeltaOp:
				old := world[mi.Flock]
				//fmt.Printf("%s heart: old=%s new=%s genghis=%s stable=%s\n", mi.T.Format(flock.Atime), old.String(), mi.String(), genghis, tstable.Format(flock.Atime))
				if old.SString() == mi.SString() {
					if old.SString() == genghis {
						if mi.T.After(tstable) {
							fs := Status{
								T:      time.Now().UTC(),
								Period: mi.T.Sub(tstart),
								Stable: true,
								Name:   genghis,
								N:      mi.N + 1,
							}
							stat <- fs
							tstable = mi.T.Add(100 * time.Minute) // prevent premature misfire
						}
					}
					continue
				}
				log.Log("flock", mi.Flock, "oldmem", old.N+1, "newmem", mi.N+1, "membership change")
				//fmt.Printf("%s %s: %d -> %d members\n", mi.T.Format(flock.Atime), mi.Flock, old.N+1, mi.N+1)
				world[mi.Flock] = mi
				if (mi.Op == crux.LeaderHeartOp) && (len(reboot) == 0) {
					genghis = mi.SString()
					tstart = mi.T
					tstable = mi.T.Add(stableT)
					log.Log(fmt.Sprintf("starting stable test %s", genghis))
				}
			case crux.ProbeOp:
				//fmt.Printf("%s %s: %d probes\n", mi.T.Format(flock.Atime), mi.Moniker, mi.N)
			case crux.RebootOp:
				reboot[mi.Moniker] = true
				log.Log("node", mi.Moniker, "nreboot", len(reboot), "rebooting")
			case crux.HeartBeatOp:
				xx, _ := json.Marshal(mi)
				log.Log("focus", "hb", string(xx))
			case crux.ExitOp:
				log.Log(nil, "got an exit")
				break loop
			}
			// prune out old entries
			cutoff := mi.T.Add(time.Duration(-1.5 * float32(stableT)))
			for k, m := range world {
				if m.T.Before(cutoff) {
					delete(world, k)
				}
			}
		}
	}
	stat <- Status{N: 0}
}

// UDP is a helper functions for a simple UDP client/server framework for the Status functions.
type UDP struct {
	port    int
	host    string
	ip      net.IP
	key     Key
	conn    net.Conn
	Inbound chan []byte
}

// NewUDPX is the per-node struct for communicating MonInfo's.
func NewUDPX(port int, host string, key Key, listen bool) (*UDP, *crux.Err) {
	un := UDP{port: port, host: host, key: key}
	if listen {
		ip, err := getIP(host)
		if err != nil {
			return nil, crux.ErrF("bad ip string '%s' (%s)", host, err.Error())
		}
		un.ip = ip
		un.Inbound = make(chan []byte, 99)
		go un.listener()
	}
	err := un.newConn()
	if err == nil {
		fmt.Printf("NewUDPX dial succeeded! (addr=%+v)\n", un.ip)
	} else {
		fmt.Printf("NewUDPX dial deferred: %s\n", err.String())
	}
	return &un, nil
}

// newConn : make an outgoing udp connection, failure ok
func (k *UDP) newConn() *crux.Err {
	ip, err := getIP(k.host)
	if err != nil {
		clog.Log.Log(nil, "UDPX getIP() failed: %s", err.Error())
		return err
	}
	k.ip = ip
	addr := net.UDPAddr{
		Port: k.port,
		IP:   k.ip,
	}
	conn, err1 := net.DialUDP("udp", nil, &addr)
	if err1 != nil {
		clog.Log.Log(nil, "UDPX DialUDP(%v) failed: %s", addr, err1.Error())
		return crux.ErrE(err1)
	}
	k.conn = conn
	return nil
}

func (k *UDP) listener() {
	addr := net.UDPAddr{
		Port: k.port,
		IP:   k.ip,
	}
	fmt.Printf("listener(%+v)\n", addr)
	ser, err := net.ListenUDP("udp", &addr)
	if err != nil {
		clog.Log.Logi(nil, "UDP listener(%s:%d) failed1: %s\n", k.ip.String(), k.port, err.Error())
		return
	}
	fmt.Printf("reading from statusListener\n")
	for {
		p := make([]byte, 9999)
		n, _, err := ser.ReadFrom(p) // 2nd arg was a
		if err != nil {
			clog.Log.Logi(nil, "UDP listener(%s:%d) read failed2: %s\n", k.ip.String(), k.port, err.Error())
			continue
		}
		//clog.Log.Log(nil, "read pkt from %s", a.String())
		//clog.Log.Log(nil, "inbound = %p (data=%s)", k.Inbound, p[:n])
		k.Inbound <- p[:n]
		//clog.Log.Log(nil, "inbound = %p", k.Inbound)
	}
}

// sendBits : send the marshaled, encrypted MonInfo
func (k *UDP) sendBits(data []byte) *crux.Err {
	if k.conn == nil {
		if err := k.newConn(); err != nil {
			return err
		}
	}
	n, err := k.conn.Write(data)
	if err != nil {
		return crux.ErrE(err)
	}
	if n != len(data) {
		return crux.ErrF("wrote %d, got %d", len(data), n)
	}
	return nil
}

// Send sends a packet.
func (k *UDP) Send(info *crux.MonInfo, key *Key) {
	fmt.Printf("kSend(%+v)\n", *info)
	bits, err := json.Marshal(*info)
	if err != nil {
		clog.Log.Log(nil, "UDPX Send/Marshal failed:", err.Error())
		return
	}
	ti, err := Encrypt(bits, key)
	if err != nil {
		clog.Log.Log(nil, "UDPX Send/Encrypt failed:", err.Error())
		return
	}
	err1 := k.sendBits(ti)
	if err1 == nil {
		return
	}
	clog.Log.Log(nil, "UDPX sendBits failed:", err1.Error())
	time.Sleep(10 * time.Millisecond)
	clog.Log.Log("resetting and retrying")
	k.conn = nil
	err1 = k.sendBits(ti)
	if err1 == nil {
		return
	}
	clog.Log.Log(nil, "UDPX sendBits failed on retry:", err1.Error())
}

// Recv receives a packet.
func (k *UDP) Recv(f []byte) *crux.MonInfo {
	KC := KCMax
	var res []byte
	var err error
	res, err = Decrypt(f, &k.key)
	if err == nil {
		KC = KCData
	} else {
		clog.Log.Log(nil, "pktdrop me %40.40s (key=%s)", f, k.key.String())
	}
	var ff crux.MonInfo
	if err == nil {
		err = json.Unmarshal(res, &ff)
		if err != nil {
			KC = KCUnmarshal
		}
	}
	//clog.Log.Log(nil, "recv state = %d", int(KC))
	crux.Assert(KC < KCMax)
	if err != nil {
		return nil
	}
	return &ff
}
