// (c) Ericsson AB 2018 All Rights Reserved.
// Contributors:
//      Christopher W. V. Hogue

package steward

import (
	"fmt"
	"time"

	"golang.org/x/net/context"

	pb "github.com/erixzone/crux/gen/cruxgen"
	"github.com/erixzone/crux/pkg/clog"
	c "github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/grpcsig"
	"github.com/erixzone/crux/pkg/idutils"
	rv "github.com/erixzone/crux/pkg/reeve"
	rb "github.com/erixzone/crux/pkg/registrydb"
)

// Fanout - the steward fanout internals - event loop that distributes keys & endpoints
type Fanout struct {
	logger    clog.Logger
	intervals chan rb.StateClock
	done      chan bool
	timetaken time.Duration
	reeves    *[]pb.EndpointInfo // Current gather of reeves
	cathorde  *[]rb.CatHorde
}

// StartNewFanout - starts a new steward fanout engine
func StartNewFanout(esttime time.Duration, logger clog.Logger) *Fanout {
	fanout := new(Fanout)
	fanout.logger = logger
	fanout.intervals = make(chan rb.StateClock)
	fanout.done = make(chan bool)
	fanout.timetaken = esttime
	go fanout.Events()
	return fanout
}

// Quit - stops the clock and Event loop goroutines
func (f *Fanout) Quit() {
	f.done <- true
}

// NewInterval - pushes a completed StateClock interval into the event loop
func (f *Fanout) NewInterval(sc rb.StateClock) {
	f.intervals <- sc
}

// Events - event loop for Fanout
func (f *Fanout) Events() {
	pidstr, ts := grpcsig.GetPidTS()
	f.logger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, "steward fanout events started")
	for {
		select {
		case <-f.done:
			close(f.intervals)
			pidstr, ts := grpcsig.GetPidTS()
			f.logger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, "steward fanout events stopped")
			return
		case interval := <-f.intervals: // ClockState arrival
			msg2 := fmt.Sprintf("Fanout started for state %d", interval.State)
			pidstr, ts := grpcsig.GetPidTS()
			f.logger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, msg2)
			ferr := f.Fanout(interval)
			if ferr != nil {
				msg3 := fmt.Sprintf("Fanout error in state %d", interval.State)
				pidstr, ts = grpcsig.GetPidTS()
				f.logger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msg3)
			}
			msg4 := fmt.Sprintf("Fanout ended for state %d", interval.State)
			pidstr, ts = grpcsig.GetPidTS()
			f.logger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, msg4)
		default:
			// do nothing
		}
	}
}

func reeveInReeveActions(nid idutils.NetIDT, ras *rb.ReeveActions) bool {
	if ras != nil {
		for _, reeve := range ras.Reeves {
			if nid.Principal == reeve.Principal {
				return true
			}
		}
	}
	return false
}

// Fanout - Does the Fanout
func (f *Fanout) Fanout(clock rb.StateClock) *c.Err {
	starttime := time.Now()
	// Do the Catalog first
	var cerr *c.Err
	f.cathorde, cerr = rb.GatherCatalog(DB)
	if cerr != nil {
		return c.ErrF("database error in Fanout GatherCatalog : %v", cerr)
	}

	// Gather Reeves - where we will fan out to...
	f.reeves, cerr = rb.GatherReeves(DB)
	if cerr != nil {
		return c.ErrF("database error in Fanout GatherReeves : %v", cerr)
	}

	tus, err := rb.UpdateOnTick(DB, clock.State)
	if err != nil {
		return c.ErrF("UpdateOnTick failed in Fanout clock state %d : %v", int(clock.State), err)
	}
	// Reeve Merge - we gather all updates for each reeve,
	// so there is only 1 grpc client, 3 calls to do everything
	// WlUpdate, EpUpdate, UpdateCatalog (if dirty)
	for _, reeve := range *f.reeves { // gather updates for each reeeve
		nidstr := reeve.Netid
		nid, _ := idutils.NetIDParse(nidstr)
		nod, _ := idutils.NodeIDParse(reeve.Nodeid)
		ra := rb.ReeveActions{}  // repackage in an individual structure
		rad := rb.ReeveActions{} // repackage del actions in an individual structure
		ra.Reeves = append(ra.Reeves, nid)
		rad.Reeves = append(rad.Reeves, nid)
		for _, update := range tus.Updates { // go through all the updates
			if reeveInReeveActions(nid, update.Oldendpoints) {
				ra.Epinfo = append(ra.Epinfo, update.Oldendpoints.Epinfo...)
				ra.Keys = append(ra.Keys, update.Oldendpoints.Keys...)
			}
			if reeveInReeveActions(nid, update.Oldclients) {
				ra.Epinfo = append(ra.Epinfo, update.Oldclients.Epinfo...)
				ra.Keys = append(ra.Keys, update.Oldclients.Keys...)
			}
			if reeveInReeveActions(nid, update.Newendpoints) {
				ra.Epinfo = append(ra.Epinfo, update.Newendpoints.Epinfo...)
				ra.Keys = append(ra.Keys, update.Newendpoints.Keys...)
			}
			if reeveInReeveActions(nid, update.Newclients) {
				ra.Epinfo = append(ra.Epinfo, update.Newclients.Epinfo...)
				ra.Keys = append(ra.Keys, update.Newclients.Keys...)
			}
			// This needs to be kept separate!!
			if reeveInReeveActions(nid, update.Delendpoints) {
				rad.Epinfo = append(ra.Epinfo, update.Delendpoints.Epinfo...)
				rad.Keys = append(ra.Keys, update.Delendpoints.Keys...)
			}
			if reeveInReeveActions(nid, update.Delclients) {
				rad.Epinfo = append(ra.Epinfo, update.Delclients.Epinfo...)
				rad.Keys = append(ra.Keys, update.Delclients.Keys...)
			}
		}
		pidstr, ts := grpcsig.GetPidTS()
		msg4 := fmt.Sprintf("ReeveActions - add %d Epinfo %d Keys %v", len(ra.Epinfo), len(ra.Keys), ra)
		f.logger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, msg4)
		msg5 := fmt.Sprintf("ReeveActions - delete %d Epinfo %d Keys %v", len(rad.Epinfo), len(rad.Keys), rad)
		f.logger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, msg5)
		uerr := UpdateReeve(nid, nod.HordeName, clock.State, &ra, &rad, f)
		if uerr != nil {
			pidstr, ts = grpcsig.GetPidTS()
			msg6 := fmt.Sprintf("ReeveActions - UpdateReeve failed for reeve at %s : %v", nid.String(), uerr)
			f.logger.Log("SEV", "WARN", "PID", pidstr, "TS", ts, msg6)
		}
	}
	// The last elapsed fanout time is updated to actual
	f.timetaken = time.Now().Sub(starttime)
	return nil
}

// UpdateReeve - Dials the Reeve, sends the fanout updates
func UpdateReeve(nid idutils.NetIDT, hordename string, tick int32, add *rb.ReeveActions, del *rb.ReeveActions, f *Fanout) *c.Err {
	epUpdate := pb.EpList{}
	epUpdate.State = tick
	for _, e := range add.Epinfo {
		ep := pb.EpInfo{
			Nodeid:   e.Nodeid,
			Netid:    e.Netid,
			Priority: e.Priority,
			Rank:     e.Rank}
		epUpdate.Add = append(epUpdate.Add, &ep)
	}
	for _, e := range del.Epinfo {
		ep := pb.EpInfo{
			Nodeid:   e.Nodeid,
			Netid:    e.Netid,
			Priority: e.Priority,
			Rank:     e.Rank}
		epUpdate.Del = append(epUpdate.Del, &ep)
	}
	wlUpdate := pb.WlList{}
	wlUpdate.State = tick
	for _, key := range add.Keys {
		wlkey := pb.WlPubKey{}
		wlkey.Json = key
		wlUpdate.Add = append(wlUpdate.Add, &wlkey)
	}
	for _, dkey := range del.Keys {
		wlkey := pb.WlPubKey{}
		wlkey.Json = dkey
		wlUpdate.Del = append(wlUpdate.Del, &wlkey)
	}
	// Populate the catalog, rules, according to horde
	catalog := pb.CatalogList{}
	catalog.State = tick
	for _, cat := range *f.cathorde {
		if cat.Hordename == hordename {
			catitem := pb.CatalogInfo{
				Nodeid:   cat.Info.Nodeid,
				Netid:    cat.Info.Netid,
				Filename: cat.Info.Filename}
			catalog.List = append(catalog.List, &catitem)
		}
	}
	// Provide the relevant rules filtered by horde
	for _, rule := range rb.CurrentRules {
		if rule.Horde == hordename {
			ruleitem := pb.RuleInfo{
				Rule:  rule.Rule,
				Horde: rule.Horde,
				From:  rule.From,
				To:    rule.To,
				Owner: rule.Owner}
			catalog.Allowed = append(catalog.Allowed, &ruleitem)
		}
	}
	reeveapi := rv.ReeveState
	if reeveapi == nil {
		return nil
	}
	signer, serr := reeveapi.ClientSigner(rv.ReeveRev)
	if serr != nil {
		return serr
	}
	// In the test - we have no register running, and no reeves to connect to, so pass
	// on updating anything...
	if signer == nil {
		return nil
	}
	// Get the Reeve Client
	reevecli, err := rv.OpenGrpcReeveClient(nid, signer, f.logger)
	if err != nil {
		return c.ErrF("error - OpenGrpcReeveClient failed for %s : %v", nid.String(), err)
	}
	// If there is no connection - we skip this update and return to it in a later clock tick
	if !((len(epUpdate.Add) == 0) && (len(epUpdate.Del) == 0)) {
		// Update the Endpoints
		ack1, herr := reevecli.EpUpdate(context.Background(), &epUpdate)
		if herr != nil {
			return c.ErrF("error - UpdateReeve EpUpdate failed for %s : %v", nid.String(), herr)
		}
		pidstr, ts := grpcsig.GetPidTS()
		msg1 := fmt.Sprintf("reeve endpoints %s updated for tick %d", nid.String(), int(tick))
		f.logger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, msg1)
		f.logger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, "ack", fmt.Sprintf("%v", ack1))
	}

	if !((len(wlUpdate.Add) == 0) && (len(wlUpdate.Del) == 0)) {
		// Update the Whitelist - keys
		ack2, herr := reevecli.WlUpdate(context.Background(), &wlUpdate)
		if herr != nil {
			return c.ErrF("error - UpdateReeve WlUpdate failed for %s : %v", nid.String(), herr)
		}
		pidstr, ts := grpcsig.GetPidTS()
		msg2 := fmt.Sprintf("reeve clients %s updated for tick %d", nid.String(), int(tick))
		f.logger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, msg2)
		f.logger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, "ack", fmt.Sprintf("%v", ack2))
	}

	// Update the Catalog and Rules (always)
	ack3, cerr := reevecli.UpdateCatalog(context.Background(), &catalog)
	if cerr != nil {
		return c.ErrF("error - UpdateReeve UpdateCatalog failed for %s : %v", nid.String(), cerr)
	}
	pidstr, ts := grpcsig.GetPidTS()
	f.logger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, "ack", fmt.Sprintf("%v", ack3))
	msg3 := fmt.Sprintf("reeve catalog, rules %s updated for tick %d", nid.String(), int(tick))
	f.logger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, msg3)
	rv.CloseGrpcReeveClient()
	return nil
}
