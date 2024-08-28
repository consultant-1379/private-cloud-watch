package horde

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/kv"
	//	"github.com/erixzone/crux/pkg/logger"
)

const (
	nodePrefix = "node/"
)

// AdminKV supports a KV-based implementation of the Administer interface.
type AdminKV struct {
	name        string
	description string
	kv          kv.KV
}

// NewAdministerKV initialises an AdminKV.
func NewAdministerKV(name, description string, kv kv.KV) *AdminKV {
	a := AdminKV{name: name, description: description, kv: kv}
	return &a
}

// UniqueID returns the horde's unique identifier.
func (kva *AdminKV) UniqueID() string {
	return kva.name
}

// Description returns the horde's description (field).
func (kva *AdminKV) Description() string {
	return kva.description
}

// RegisterNode registers a node.
func (kva *AdminKV) RegisterNode(name string, tags []string) *crux.Err {
	n := Node{Name: name, Tags: tags}
	b, e := json.Marshal(&n)
	if e != nil {
		return crux.ErrE(e)
	}
	er := kva.kv.Put(nodePrefix+name, string(b))
	return er

}

// Nodes returns a list of the live Nodes.
func (kva *AdminKV) Nodes() ([]Node, *crux.Err) {
	var nodes []Node
	names := kva.kv.GetKeys(nodePrefix)
	sort.Strings(names)
	for _, nn := range names {
		b, e := kva.kv.Get(nn)
		if e != nil {
			return nil, e
		}
		var nd Node
		if ue := json.Unmarshal([]byte(b), &nd); ue != nil {
			return nil, crux.ErrE(ue)
		}
		nodes = append(nodes, nd)
	}
	return nodes, nil
}

func (n Node) String() string {
	return fmt.Sprintf("%s(%s)", n.Name, strings.Join(n.Tags, ","))
}
