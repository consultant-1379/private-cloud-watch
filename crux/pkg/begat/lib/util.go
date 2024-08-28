package lib

import (
	"fmt"
	"strings"
)

func (v Variable) String() string {
	return fmt.Sprintf("VAR %s: attr=%d val=%s", v.Name, v.Attr, v.Val)
}

func prettyList(sl []string) string {
	return "[" + strings.Join(sl, ", ") + "]"
}

func (s *Statement) String() string {
	res := "stmt "
	switch s.What {
	case StatementVar:
		res += fmt.Sprintf("var %s = %s", s.Vr.Name, s.Vr.Val)
	case StatementCallFunc:
		res += fmt.Sprintf("callfunc %s(%s)", s.Name, prettyList(s.Args))
	case StatementCallDict:
		res += fmt.Sprintf("calldict %s(%s) %s", s.Name, prettyList(s.Args), s.Dict)
	case StatementDict:
		res += fmt.Sprintf("dictum %s", s.Dict)
	case StatementApply:
		res += fmt.Sprintf("apply %s to %s", s.Args[0], prettyList(s.Args[1:]))
	case StatementFunc:
		res += fmt.Sprintf("func %s{}", s.Fn.Name)
	case StatementCd:
		res += fmt.Sprintf("cd %s {%s}", s.Dir, s.Block.String())
	case StatementMount:
		res += fmt.Sprintf("mount(%s)", prettyList(s.Args))
	}
	return res
}

func (b Block) String() string {
	s := "??block??"
	switch len(b.Stmts) {
	case 0:
		s = ""
	case 1:
		s = b.Stmts[0].String()
	default:
		s = b.Stmts[0].String() + " ... " + b.Stmts[len(b.Stmts)-1].String()
	}
	return s
}

func (d *Dictum) String() string {
	name := d.Name
	if name == "" {
		name = "<anon>"
	}
	name += "[" + d.Src + "]"
	for _, x := range d.Inputs {
		name += fmt.Sprintf(" i=%s", x)
	}
	for _, x := range d.InEnts {
		name += fmt.Sprintf(" I=%s", x.Name)
	}
	for _, x := range d.Outputs {
		name += fmt.Sprintf(" o=%s", x)
	}
	return name
}

// Nom does a prettyprint of a Dictum.
func (d *Dictum) Nom() string {
	name := d.Name
	if name == "" {
		name = "<anon>"
	}
	name += "[" + d.Src + "]:"
	for _, x := range d.Outputs {
		name += fmt.Sprintf(" o=%s", x)
	}
	return name
}

func (e *Ent) String() string {
	return fmt.Sprintf("Ent{%s status=%s hash=%d}", e.Name, e.Status, e.Hash)
}

func (c *Chore) prOuts() string {
	return strings.Join(c.D.Outputs, " ")
}

func (c *Chore) prDict() string {
	return fmt.Sprintf("%s [%s]", c.D.Name, c.D.Src)
}

const ssep = "|"

func (r *Recipe) String() string {
	return r.Interp + ssep + string(r.Recipe)
}

// this needs a lot more work; TBD
func getSig(ch *Chore) string {
	ret := ""
	for _, x := range ch.D.Inputs {
		ret = ret + ssep + x
	}
	for _, x := range ch.D.Outputs {
		ret = ret + ssep + x
	}
	ret += ssep + (&ch.D.Recipe).String()
	// more here to be sure, and don't forget to include checksums
	if ret == "" {
		return ""
	}
	return ret[1:]
}
