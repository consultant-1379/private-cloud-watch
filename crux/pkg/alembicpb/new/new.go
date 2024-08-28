package alembicpbnew

import (
	"strings"

	ab "github.com/erixzone/crux/gen/alembicpb"
	"github.com/erixzone/crux/pkg/alembicpb"
)

// this is an odd arrangement of source forced by the rule that
// generated code lives under stix/gen. given that rule,
// the natural code layout leads to an import cycle. the only
// plausible way to avoid that is alembicgen stuff can only be invoked
// by code in a (not alembic) package. we can localise that to just
// the New function, so here we are.

// New populates a Alembic from a configuration string.
func New(filename, input string, noisy bool) (*alembicpb.ProtoPkg, error) {
	rdr := strings.NewReader(input)
	// nex creates the function: func NewLexerWithInit(in io.Reader, initFun func(*Lexer)) *Lexer
	// go tool yacc creates the function: func yyParse(yylex yyLexer) int
	a, err := ab.Parse(filename, rdr, noisy)
	if err != nil {
		return nil, err
	}
	return a, nil
}
