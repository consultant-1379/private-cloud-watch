package horde

import (
	"github.com/erixzone/crux/pkg/clog"
	"github.com/erixzone/crux/pkg/crux"
)

// ActionMem is our internal action interface
type ActionMem struct {
	logger clog.Logger
	svcs   []Service
}

// NewActionMem implements a simple KV based aAction interface
func NewActionMem(logger clog.Logger) *ActionMem {
	return &ActionMem{logger: logger, svcs: make([]Service, 0)}
}

// Start starts up services
func (a *ActionMem) Start(node, service string, count int) {
	for count > 0 {
		count--
		s := Service{UniqueID: crux.SmallID(), Name: service, Node: node, Stage: StageReady}
		a.logger.Log(nil, "starting %s on node %s (id=%s)", s.Name, s.Node, s.UniqueID)
		a.svcs = append(a.svcs, s)
	}
}

// Start1 starts up a single service on a node
func (a *ActionMem) Start1(node, service, addr string) {
	s := Service{UniqueID: crux.SmallID(), Name: service, Node: node, Addr: addr, Stage: StageReady}
	a.logger.Log(nil, "starting %s on node %s at addr %s (id=%s)", s.Name, s.Node, s.Addr, s.UniqueID)
	a.svcs = append(a.svcs, s)
}

// Reset resets something
func (a *ActionMem) Reset() {
	// nada for now
}

// Stop stops services
func (a *ActionMem) Stop(node, service string, count int) {
	// go thru and swap deletables to end
	var e = len(a.svcs)
oloop:
	for count > 0 {
		count--
		for i := 0; i < len(a.svcs); i++ {
			s := a.svcs[i]
			if (s.Node == node) && (s.Name == service) {
				// eliminate me
				a.logger.Log(nil, "stopping %s on node %s (id=%s)", s.Name, s.Node, s.UniqueID)
				e--
				a.svcs[e], a.svcs[i] = a.svcs[i], a.svcs[e]
				i--
				continue oloop
			}
		}
	}
	a.svcs = a.svcs[:e]
}

// What returns whose up
func (a *ActionMem) What() []Service {
	return a.svcs
}
