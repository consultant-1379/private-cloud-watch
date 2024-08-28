package lib

import (
	"github.com/erixzone/crux/pkg/begat/common"
)

// Statement is the piece parts of a Dictum.
type Statement struct {
	What StatementType
	// essentially a union below
	Vr    Variable
	Fn    *Func
	Dict  *Dictum
	Name  string
	Args  []string
	Dir   string
	Mount []*Statement
	Block *Block
}

// Block is a bunch of statements.
type Block struct {
	Stmts []*Statement
}

// Variable represents an up or down inherited varibale.
type Variable struct {
	Name string
	Attr uint // attributes
	Val  string
}

// Ent represents a file.
type Ent struct {
	Status EntType
	Name   string
	Depend bool
	Hash   common.Hash
}

// Recipe represents a recipe.
type Recipe struct {
	Interp string // what program executes this as stdin
	Recipe []byte
}

// Dictum is a procedure equivalent.
type Dictum struct {
	Name      string
	LogicalID string // for testing ...
	Src       string
	Attr      uint
	Args      []string
	Imports   []string
	UpVars    []Variable
	DownVars  []Variable
	Inputs    []string
	Outputs   []string
	Depends   []string
	InEnts    []*Ent // eliminate
	OutEnts   []*Ent // eliminate
	Recipe
}

// Func represents a begat function.
type Func struct {
	Name  string
	Args  []string
	Stmts []*Statement
}

// Parse  represents the total output from a parse
type Parse struct {
	dicts []*Dictum
	stmts []*Statement
	code  []*Statement
}
