package lib

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"
)

func newParse() *Parse {
	p := Parse{}
	p.stmts = make([]*Statement, 0)
	p.dicts = make([]*Dictum, 0)
	p.code = make([]*Statement, 0)
	return &p
}

// ParseFile parse a complete file
func ParseFile(pathname string) (*Parse, error) {
	lex := lexFile(pathname)
	parse := newParse()
	// temporary thing; just strip whitespace
	ch := make(chan Token)
	go stripWhitespace(lex.tokens, ch)
	err := parse.begatfile(ch)
	if err != nil {
		return parse, err
	}
	err = parse.instantiate()
	if err != nil {
		return parse, err
	}
	return parse, nil
}

func stripWhitespace(inbound chan Token, outbound chan Token) {
	for {
		t := <-inbound
		switch t.what {
		case tokenNL, tokenWS:
			// nothing
		case tokenEOF:
			outbound <- t
			return
		default:
			outbound <- t
		}
	}
}

// parse a whole file
func (p *Parse) begatfile(tc chan Token) error {
	for {
		s, err := parseStatement(tc, tokenEOF)
		if err != nil {
			drain(tc)
			return err
		}
		if s == nil {
			break
		}
		p.stmts = append(p.stmts, s)
	}
	return nil
}

// parse a Statement
func parseStatement(tc chan Token, last tokenType) (*Statement, error) {
	tok := <-tc
	if tok.what == last {
		return nil, nil
	}
	//	fmt.Printf("parse statement: %s\n", tok)
	if tok.what == tokenWord {
		switch string(tok.val) {
		case "var":
			v, err := parseVar(tc)
			if err != nil {
				return nil, err
			}
			s := newStatement(StatementVar)
			s.Vr = v
			return s, nil
		case "dictum":
			src := fmt.Sprintf("%s:%d", tok.file, tok.line)
			d, err := parseDictum(tc)
			d.Src = src
			if err != nil {
				return nil, err
			}
			s := newStatement(StatementDict)
			s.Dict = &d
			return s, nil
		case "func":
			f, err := parseFunc(tc)
			if err != nil {
				return nil, err
			}
			s := newStatement(StatementFunc)
			s.Fn = f
			return s, nil
		case "apply":
			args, err := parseSet(tc, tokenLpar, tokenRpar, false)
			if err != nil {
				return nil, err
			}
			s := newStatement(StatementApply)
			s.Args = args
			return s, nil
		case "cd":
			tok = <-tc
			s := newStatement(StatementCd)
			if tok.what != tokenWord {
				return nil, parseError(tok, "word")
			}
			var err error
			s.Dir = string(tok.val)
			s.Block, err = parseBlock(tc)
			return s, err
		case "run":
			tok = <-tc
			s := newStatement(StatementCallDict)
			s.Name = string(tok.val)
			tok = <-tc
			if tok.what == tokenLpar {
				args, err := parseSet(tc, tokenError, tokenRpar, false)
				if err != nil {
					return nil, err
				}
				s.Args = args
			} else {
				s.Args = nil
			}
			return s, nil
		case "mount":
			s := newStatement(StatementMount)
			args, err := parseStringSet(tc, tokenLpar, tokenRpar, false)
			if err != nil {
				return nil, err
			}
			s.Args = args
			return s, nil
		default:
			s := newStatement(StatementCallFunc)
			s.Name = string(tok.val)
			args, err := parseSet(tc, tokenLpar, tokenRpar, false)
			if err != nil {
				return nil, err
			}
			s.Args = args
			return s, nil
		}
	}
	return nil, parseError(tok, "Statement")
}

// parse a variable declaration
func parseVar(tc chan Token) (Variable, error) {
	var v Variable
	var err error

	tok := <-tc
	//	if isName(tok) {
	//		v.Name = string(tok.val)
	//	} else {
	//		return v, parseError(tok, "variable name")
	//	}
	v.Name = string(tok.val)
	tok = <-tc
	if tok.what == tokenLSB {
		v.Attr, err = parseAttr(false, tc)
		if err != nil {
			return v, err
		}
		tok = <-tc
	}
	if tok.what != tokenEqual {
		return v, parseError(tok, "=")
	}
	tok = <-tc
	if (tok.what == tokenWord) || (tok.what == tokenString) {
		v.Val = string(tok.val)
	}
	return v, nil
}

// parse and attribute section
func parseAttr(absorbLB bool, tc chan Token) (uint, error) {
	tok := <-tc
	if absorbLB {
		if tok.what != tokenLSB {
			return 0, parseError(tok, "[")
		}
		tok = <-tc
	}
	if (tok.what == tokenString) || (tok.what == tokenWord) {
		// process tok.val
	} else {
		return 0, parseError(tok, "word or string")
	}
	tok = <-tc
	if tok.what != tokenRSB {
		return 0, parseError(tok, "]")
	}
	return 13, nil
}

// parse a dictum
func parseDictum(tc chan Token) (Dictum, error) {
	var d Dictum
	var err error

	tok := <-tc
	if isName(tok) {
		d.Name = string(tok.val)
		tok = <-tc
	} else {
		return d, parseError(tok, "name")
	}
	if tok.what == tokenLpar {
		d.Args, err = parseSet(tc, tokenError, tokenRpar, true)
		if err != nil {
			return d, err
		}
		tok = <-tc
	}
	if tok.what != tokenLbrace {
		return d, parseError(tok, "{")
	}
	for {
		tok = <-tc
		if tok.what == tokenRbrace {
			break
		}
		if tok.what == tokenWord {
			var err error

			switch string(tok.val) {
			case "attr":
				d.Attr, err = parseAttr(true, tc)
			case "import":
				d.Imports, err = parseSet(tc, tokenLSB, tokenRSB, true)
			case "up":
				d.UpVars, err = parseVarList(tc)
			case "down":
				d.DownVars, err = parseVarList(tc)
			case "in":
				d.Inputs, err = parseEntList(tc)
			case "out":
				d.Outputs, err = parseEntList(tc)
			case "exec":
				tok = <-tc
				if tok.what == tokenLSB {
					tok = <-tc
					if tok.what != tokenWord {
						return d, parseError(tok, "exec word")
					}
					d.Recipe.Interp = string(tok.val)
					tok = <-tc
					if tok.what != tokenRSB {
						return d, parseError(tok, "]")
					}
					tok = <-tc
				} else {
					d.Recipe.Interp = "/bin/sh" // fix me later ($BEGATSHELL)
				}
				if tok.what != tokenVerbatim {
					return d, parseError(tok, "verbatim")
				}
				d.Recipe.Recipe = make([]byte, len(tok.val))
				copy(d.Recipe.Recipe, tok.val)
			default:
				err = parseError(tok, "one of attr,up,down,in,out,exec")
			}
			if err != nil {
				return d, err
			}
		} else {
			return d, parseError(tok, "one of attr,up,down,in,out,exec")
		}
	}
	return d, nil
}

// parse a function
func parseFunc(tc chan Token) (*Func, error) {
	var f Func
	var err error

	tok := <-tc
	if isName(tok) {
		f.Name = string(tok.val)
	} else {
		return nil, parseError(tok, "name")
	}
	f.Args, err = parseSet(tc, tokenLpar, tokenRpar, true)
	if err != nil {
		return nil, err
	}
	tok = <-tc
	if tok.what != tokenLbrace {
		return nil, parseError(tok, "{")
	}
	f.Stmts = make([]*Statement, 0)
	for {
		s, err := parseStatement(tc, tokenRbrace)
		if err != nil {
			return nil, err
		}
		if s == nil {
			break
		}
		f.Stmts = append(f.Stmts, s)
	}
	return &f, nil
}

// parse a block
func parseBlock(tc chan Token) (*Block, error) {
	var b Block

	tok := <-tc
	if tok.what != tokenLbrace {
		return nil, parseError(tok, "{")
	}
	b.Stmts = make([]*Statement, 0)
	for {
		s, err := parseStatement(tc, tokenRbrace)
		if err != nil {
			return nil, err
		}
		if s == nil {
			break
		}
		b.Stmts = append(b.Stmts, s)
	}
	return &b, nil
}

// parse square bracket list
func parseVarList(tc chan Token) ([]Variable, error) {
	tok := <-tc
	if tok.what != tokenLSB {
		return nil, parseError(tok, "[")
	}
	tok = <-tc
	vars := make([]Variable, 0)
	for {
		if tok.what == tokenRSB {
			break
		}
		if tok.what != tokenWord {
			return nil, parseError(tok, "word")
		}
		if !isName(tok) {
			return nil, parseError(tok, "name")
		}
		v := Variable{Name: string(tok.val)}
		tok = <-tc
		if tok.what == tokenEqual {
			tok = <-tc
			if (tok.what == tokenString) || (tok.what == tokenWord) {
				v.Val = string(tok.val)
			} else {
				return nil, parseError(tok, "word or string")
			}
			tok = <-tc
		}
		vars = append(vars, v)
	}
	return vars, nil
}

// parse bracketed list
func parseSet(tc chan Token, beg tokenType, end tokenType, name bool) ([]string, error) {
	var tok Token
	if beg != tokenError {
		tok = <-tc
		if tok.what != beg {
			return nil, parseError(tok, beg.String())
		}
	}
	tok = <-tc
	words := make([]string, 0)
	for {
		if tok.what == end {
			break
		}
		if tok.what != tokenWord {
			return nil, parseError(tok, "word")
		}
		if name && !isName(tok) {
			return nil, parseError(tok, "name")
		}
		words = append(words, string(tok.val))
		tok = <-tc
	}
	return words, nil
}

// parse bracketed string list
func parseStringSet(tc chan Token, beg tokenType, end tokenType, name bool) ([]string, error) {
	var tok Token
	if beg != tokenError {
		tok = <-tc
		if tok.what != beg {
			return nil, parseError(tok, beg.String())
		}
	}
	tok = <-tc
	words := make([]string, 0)
	for {
		if tok.what == end {
			break
		}
		if tok.what != tokenString {
			return nil, parseError(tok, "word")
		}
		words = append(words, string(tok.val))
		tok = <-tc
	}
	return words, nil
}

// parse square bracket list
func parseEntList(tc chan Token) ([]string, error) {
	tok := <-tc
	if tok.what != tokenLSB {
		return nil, parseError(tok, "[")
	}
	ents := make([]string, 0)
	for {
		tok = <-tc
		if tok.what == tokenRSB {
			break
		}
		if tok.what != tokenWord {
			return nil, parseError(tok, "word")
		}
		ents = append(ents, string(tok.val))
	}
	return ents, nil
}

// handle a common error
func parseError(t Token, wanted string) error {
	//    panic(fmt.Errorf("error: got %s; wanted %s", t, wanted))
	return fmt.Errorf("error: got %s; wanted %s", t, wanted)
}

func soName(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || (r == '_')
}

func notName(r rune) bool {
	return !soName(r)
}

// is the Token a variable name?
func isName(t Token) bool {
	if t.what != tokenWord {
		return false
	}
	return strings.IndexFunc(string(t.val), notName) == -1
}

// drain the Token channel so that everything cleans up
func drain(tc chan Token) {
	for {
		t := <-tc
		if t.what == tokenEOF {
			return
		}
	}
}

func (p *Parse) instantiate() error {
	fns := make(map[string]*Func, 0)
	dicts := make(map[string]*Dictum, 0)
	vars := make(map[string]*Variable, 0)
	gmnt := make([]*Statement, 0)
	lmnt := make([]*Statement, 0)
	// first, absorb defns
	for _, s := range p.stmts {
		if s.What == StatementFunc {
			fns[s.Fn.Name] = s.Fn
		}
		if s.What == StatementDict {
			dicts[s.Dict.Name] = s.Dict
		}
	}
	// add predefined variables
	predefinedVars(vars)
	// lay them down
	err := p.sset(p.stmts, fns, dicts, vars, nil, gmnt, lmnt, ".")
	//	fmt.Printf("instantiate returned %d code\n", len(p.code))
	if err == nil {
		idx := 1
		for i := range p.code {
			if p.code[i].What == StatementCallDict {
				p.code[i].Dict.LogicalID = fmt.Sprintf("dict%04d", idx)
				idx++
			}
		}
	}
	return err
}

// do a slice of Statements
func (p *Parse) sset(ss []*Statement, fns map[string]*Func, dicts map[string]*Dictum, gvars, lvars map[string]*Variable, gmount, lmount []*Statement, dir string) error {
	for _, s := range ss {
		switch s.What {
		case StatementVar:
			ns := newStatement(s.What)
			ns.Vr = s.Vr
			ns.Vr.Name = expand(gvars, lvars, nil, s.Vr.Name)
			ns.Vr.Val = expand(gvars, lvars, nil, s.Vr.Val)
			//p.code = append(p.code, ns)
			gvars[ns.Vr.Name] = &ns.Vr
		case StatementCallDict:
			err := p.calldict(s.Name, s.Args, dicts, gvars, lvars, gmount, lmount, dir)
			if err != nil {
				return err
			}
			continue
		case StatementCallFunc:
			err := p.callfunc(s.Name, s.Args, fns, dicts, gvars, lvars, gmount, lmount, dir)
			if err != nil {
				return err
			}
			continue
		case StatementDict:
			// ignore as its a defn
		case StatementApply:
			// first we need to expand all the arguments
			actuals := make([]string, 0)
			for _, a := range s.Args {
				actuals = append(actuals, strings.Split(expand(gvars, lvars, nil, a), " ")...)
			}
			who := actuals[0]
			arg := make([]string, 1)
			for _, a := range actuals[1:] {
				arg[0] = a
				var err error
				if _, ok := fns[who]; ok {
					err = p.callfunc(who, arg, fns, dicts, gvars, lvars, gmount, lmount, dir)
				} else {
					err = p.calldict(who, arg, dicts, gvars, lvars, gmount, lmount, dir)
				}
				if err != nil {
					return err
				}
			}
		case StatementFunc:
			// ignore these
		case StatementCd:
			err := p.sset(s.Block.Stmts, fns, dicts, gvars, lvars, gmount, lmount, chdir(dir, s.Dir))
			if err != nil {
				return err
			}
		case StatementMount:
			ns := newStatement(s.What)
			ns.Args = s.Args
			lmount = append(lmount, ns)

		default:
			return nil // should be unknown Statement type internal error
		}
	}
	return nil
}

// call a dictum
func (p *Parse) calldict(name string, args []string, dicts map[string]*Dictum, gvars, lvars map[string]*Variable, gmount, lmount []*Statement, dir string) error {
	d, ok := dicts[name]
	if ok {
		nlvars, err := popArgs(lvars, d.Args, args)
		if err != nil {
			return err
		}
		ns := newStatement(StatementCallDict)
		ns.Dict = invoke(d, gvars, nlvars)
		ns.Dir = dir
		ns.Mount = append(ns.Mount, gmount...)
		ns.Mount = append(ns.Mount, lmount...)
		p.code = append(p.code, ns)
		return nil
	}
	return fmt.Errorf("unknown dictum %s", name)
}

// call a function
func (p *Parse) callfunc(name string, args []string, fns map[string]*Func, dicts map[string]*Dictum, gvars, lvars map[string]*Variable, gmount, lmount []*Statement, dir string) error {
	f, ok := fns[name]
	if ok {
		nlvars, err := popArgs(lvars, f.Args, args)
		if err != nil {
			return err
		}
		// make a newgmount = gmount+lmount; it will effective reset after call
		var ngm []*Statement
		ngm = append(ngm, gmount...)
		ngm = append(ngm, lmount...)
		p.sset(f.Stmts, fns, dicts, gvars, nlvars, ngm, make([]*Statement, 0), dir)
		return nil
	}
	return fmt.Errorf("unknown function %s", name)
}

func invoke(d *Dictum, vars1 map[string]*Variable, vars2 map[string]*Variable) *Dictum {
	nd := Dictum{Name: d.Name, Src: d.Src, Attr: d.Attr, Args: nil, Imports: d.Imports, Recipe: d.Recipe}
	nd.UpVars = d.UpVars[:]
	nd.DownVars = d.DownVars[:]
	if len(d.Inputs) == 0 {
		nd.Inputs = make([]string, 0)
	} else {
		x := ""
		for _, e := range d.Inputs {
			x += " " + expand(vars1, vars2, nil, e)
		}
		nd.Inputs = strings.Split(x[1:], " ")
	}
	nd.Outputs = make([]string, 0)
	for _, e := range d.Outputs {
		nd.Outputs = append(nd.Outputs, expand(vars1, vars2, nil, e))
	}
	return &nd
}

func popArgs(vm map[string]*Variable, formal []string, actual []string) (map[string]*Variable, error) {
	newm := make(map[string]*Variable, 0)
	for k, v := range vm {
		newm[k] = v
	}
	for i := range formal {
		v := Variable{Name: formal[i], Val: actual[i]}
		newm[formal[i]] = &v
	}
	return newm, nil
}
func predefinedVars(vars map[string]*Variable) {
	vs := []Variable{{Name: "NEXEC", Attr: 0, Val: "3"}}
	for _, v := range vs {
		vars[v.Name] = &v
	}
}

func expand(vars1, vars2, vars3 map[string]*Variable, val string) string {
	// really, i hate runes. or rather, they are sooo much bloody work
	res := ""
	i := 0
	for {
		j := strings.IndexRune(val[i:], '$')
		if j < 0 {
			res += val[i:]
			break
		}
		// step over $
		j++
		// check first for ${
		if val[j:j+1] == "{" {
			k, xres := expand1(val, j, vars1, vars2, vars3)
			i = k
			res += xres
			continue
		}
		// now check if it is an expandable name
		k := strings.IndexFunc(val[j:], soName)
		if k != 0 {
			// not a name, so just continue on
			res += val[i:j]
			i = j
			continue
		}
		// now check if it is an expandable name
		k = strings.IndexFunc(val[j:], notName)
		if k < 0 {
			// must be name all the way to the end of teh string
			k = len(val)
		} else {
			k += j
		}
		vname := val[j:k]
		if j > 1 {
			res += val[:j-1]
		}
		res += vval(vars1, vars2, vars3, vname)
		i = k
	}
	return res
}

// handle ${var:%.xx=s.%.y}
func expand1(val string, i int, vars1, vars2, vars3 map[string]*Variable) (int, string) {
	// horrid wretched semantics; who thought of this poxy syntax?? ohhh, me. oh my!
	k := i + 1 // i points at {
	m := strings.IndexFunc(val[k:], soName)
	if m != 0 {
		// not a name, so barf
		fmt.Printf("warning: expected a var name at %s\n", val[k:])
		return k, "XXX"
	}
	m = strings.IndexFunc(val[k:], notName)
	if m < 0 {
		fmt.Printf("warning: expected a closing } at %s\n", val[k:])
		return k, "XXX"
	}
	vname := val[k : k+m]
	k += m
	if val[k:k+1] != ":" {
		fmt.Printf("warning: expected a : at %s\n", val[k:])
		return k, "XXX"
	}
	k++
	m = strings.IndexRune(val[k:], '=')
	if m < 0 {
		fmt.Printf("warning: expected a = at %s\n", val[k:])
		return k, "XXX"
	}
	pre := val[k : k+m]
	k += m + 1
	m = strings.IndexRune(val[k:], '}')
	if m < 0 {
		fmt.Printf("warning: expected a } at %s\n", val[k:])
		return k, "XXX"
	}
	post := val[k : k+m]
	k += m + 1 // this is the return index
	// after all that, we can now actually starting doing some work. gack!
	base := strings.Split(vval(vars1, vars2, vars3, vname), " ")
	res := ""
	for _, word := range base {
		res += " " + transmogrify(word, pre, post)
	}
	if len(res) > 0 {
		res = res[1:]
	}
	return k, res
}

func transmogrify(word, pre, post string) string {
	ipre := strings.IndexRune(pre, '%')
	ipost := strings.IndexRune(post, '%')
	lword := len(word)
	lpre := len(pre)
	if (ipre < 0) || (ipost < 0) {
		// stop wasting our time
		return word
	}
	//	fmt.Printf("%d %d %d %d\n", ipre, lpre, ipost, lword)
	//	fmt.Printf("ZZ '%s' '%s' '%s' '%s'\n", word[0:ipre], pre[0:ipre], pre[ipre+1:], word[lword-(lpre-ipre-1):])
	if (word[0:ipre] != pre[0:ipre]) || (pre[ipre+1:] != word[lword-(lpre-ipre-1):]) {
		return word
	}
	//	fmt.Printf("%%='%s'\n", word[ipre:lword-(lpre-ipre-1)])
	return post[0:ipost] + word[ipre:lword-(lpre-ipre-1)] + post[ipost+1:]
}

func vval(vars1, vars2, vars3 map[string]*Variable, vname string) string {
	var v *Variable
	var ok bool
	if vars3 != nil {
		v, ok = vars3[vname]
		if ok {
			return v.Val
		}
	}
	if vars2 != nil {
		v, ok = vars2[vname]
		if ok {
			return v.Val
		}
	}
	if vars1 != nil {
		v, ok = vars1[vname]
		if ok {
			return v.Val
		}
	}
	return "$<" + vname + " not found>"
}

func (p *Parse) pretty() {
	fmt.Printf("pretty!\n")
	for i, s := range p.code {
		fmt.Printf("%d: %s\n", i, s)
	}
}

func chdir(wd string, where string) string {
	if filepath.IsAbs(where) {
		return where
	}
	wd = filepath.Clean(filepath.Join(wd, where))
	return wd
}

func newStatement(w StatementType) *Statement {
	s := Statement{What: w, Mount: make([]*Statement, 0)}
	return &s
}
