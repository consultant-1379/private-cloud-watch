package horde

import (
	"fmt"
	"strings"

	"github.com/erixzone/crux/pkg/clog"
	"github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/kv"
)

// various constants.
const (
	HordeSpecFlag = "-H"
	SpecSep       = ","
	HordeRoot     = "horde/"
)

// GetHorde returns the actual type for the supplied spec
func GetHorde(name, spec string, description string) (*Horde, error) {
	fields := strings.Split(spec, SpecSep)
	switch fields[0] {
	case "crux":
		var h Horde
		var bspec string
		switch len(fields) {
		case 1:
			bspec = ""
		case 2:
			bspec = fields[1]
		default:
			return nil, fmt.Errorf("Horde spec '%s' for crux must be crux[%sconsuladdr]", spec, SpecSep)
		}
		var err error
		if bspec == "local" {
			h.KV = kv.NewLocalKV()
		} else {
			h.KV, err = kv.NewConsulKV(bspec)
			if err != nil {
				return nil, err
			}
		}
		// if no horde name, synthesise a unique one ourselves
		if name == "" {
			name = crux.SmallID()
		}
		h.Adm = NewAdministerKV(name, description, h.KV)
		h.Act = NewActionMem(clog.Log.With("focus", "hordemngt"))
		return &h, err
	default:
		return nil, fmt.Errorf("unknown Horde type '%s'; must be one of [crux] (spec='%s')", fields[0], spec)
	}
}

/*
// Horde this is a reference implementaion of the Horder interface built on
// the Boarder interface.
type Horde struct {
	unique      string
	description string
	spec        string
	sflag       string
	b           Boarder
	g           Governor
	stage       map[string]StageStr
}
*/

// some more standard key names.
const (
	KRole    = "role"
	KPid     = "pid"
	KNode    = "node"
	KStatus  = "status"
	KCmd     = "cmd"
	KCmdNone = "none"
)

/*
// UniqueID returns the unique identifier for this horde.
func (h *Horde) UniqueID() string {
	return h.unique
}

// Description returns the given name (nickname) for this horde.
func (h *Horde) Description() string {
	return h.description
}

// SetSpec sets what Spec returns
func (h *Horde) SetSpec(spec string) {
	fmt.Printf("set horde spec to '%s'\n", spec)
	h.sflag = spec
}

// Spec returns what SetSpec set
func (h *Horde) Spec() string {
	return h.sflag
}

// helper function to only add the horde root if it is missing
func check(prefix, s string) string {
	if (len(s) > 0) && (s[:1] == "/") {
		logger.Warningf("horde boarder function key '%s' starts with /", s)
		s = s[1:]
	}
	n := len(HordeRoot)
	if (len(s) >= n) && (s[:n] == HordeRoot) {
		return s
	}
	return HordeRoot + prefix + "/" + s
}

// List returns a blackboard List with a key prefixed by horde info.
func (h *Horde) List(prefix string) ([]string, error) {
	keys, err := h.b.List(check(h.unique, prefix))
	if err != nil {
		return nil, err
	}
	prefixLen := len(check(h.unique, ""))
	for i := 0; i < len(keys); i++ {
		keys[i] = keys[i][prefixLen:]
	}
	return keys, nil
}

// Get returns a blackboard Get with a key prefixed by horde info.
func (h *Horde) Get(key string) (string, error) {
	return h.b.Get(check(h.unique, key))
}

// Put returns a blackboard list with a key prefixed by horde info.
func (h *Horde) Put(key, value string) error {
	return h.b.Put(check(h.unique, key), value)
}

// Delete returns a blackboard list with a key prefixed by horde info.
func (h *Horde) Delete(key string) error {
	return h.b.Delete(check(h.unique, key))
}

// CAS returns a blackboard list with a key prefixed by horde info.
func (h *Horde) CAS(key, ovalue, nvalue string) error {
	return h.b.CAS(check(h.unique, key), ovalue, nvalue)
}

// Leader returns the cluster blackboard leader
func (h *Horde) Leader() string {
	return h.b.Leader()
}

// DumpKV dumps the kvstore
func DumpKV(h Horder) {
	fmt.Printf("horde KV dump:\n")
	keys, err := h.List("")
	if err != nil {
		fmt.Printf("<<list error: %s>>\n", err.Error())
		fmt.Printf("-------------dump done\n")
		return
	}
	sort.Strings(keys)
	for _, key := range keys {
		val, err := h.Get(key)
		if err != nil {
			fmt.Printf("<<get error: %s>>\n", err.Error())
			return
		}
		fmt.Printf("%s: %s\n", key, val)
	}
	fmt.Printf("-------------dump done\n")
}
*/
