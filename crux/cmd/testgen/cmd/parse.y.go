//line parse.y:5
package cmd

import __yyfmt__ "fmt"

//line parse.y:5

import (
	"fmt"
	"io"
)

var begatTests []*begatTest

func init() {
	begatErrorVerbose = true
}

//line parse.y:20
type begatSymType struct {
	yys      int
	btests   []*begatTest
	btest    *begatTest
	str      string
	pre      begatPre
	post     begatPost
	thing    begatThing
	thingset []begatThing
}

const IDENTIFIER = 57346
const BEGATFILE = 57347
const HISTORY = 57348
const PASTICHE = 57349
const INPUTS = 57350
const OUTPUTS = 57351
const DICTUMS = 57352
const ARROW = 57353
const PLUS = 57354
const MINUS = 57355
const COLON = 57356
const LBRACE = 57357
const RBRACE = 57358
const LBRACK = 57359
const RBRACK = 57360
const EQUAL = 57361

var begatToknames = [...]string{
	"$end",
	"error",
	"$unk",
	"IDENTIFIER",
	"BEGATFILE",
	"HISTORY",
	"PASTICHE",
	"INPUTS",
	"OUTPUTS",
	"DICTUMS",
	"ARROW",
	"PLUS",
	"MINUS",
	"COLON",
	"LBRACE",
	"RBRACE",
	"LBRACK",
	"RBRACK",
	"EQUAL",
}
var begatStatenames = [...]string{}

const begatEofCode = 1
const begatErrCode = 2
const begatInitialStackSize = 16

//line parse.y:127

// FlexLex lets us insert golex style lexical parsers
type FlexLex struct {
	lval *begatSymType
}

// Lex is what the parser needs from lexing
func (fl *FlexLex) Lex(lv *begatSymType) int {
	ret := yylex()
	*lv = lval
	return ret
}

func (fl *FlexLex) Error(e string) {
	fmt.Printf("error:%s:%d: %s\n", yyfile, yylineno, e)
}

func begat(filename string, rdr io.Reader, noisy bool) error {
	if noisy {
		begatDebug = 4
	} else {
		begatDebug = 0
	}
	var fl FlexLex
	yyin = rdr
	yyfile = filename
	ret := begatParse(&fl)
	if ret == 1 {
		return fmt.Errorf("parse failed")
	}
	return nil
}

//line yacctab:1
var begatExca = [...]int{
	-1, 1,
	1, -1,
	-2, 0,
}

const begatPrivate = 57344

const begatLast = 47

var begatAct = [...]int{

	27, 23, 24, 40, 13, 14, 15, 16, 32, 29,
	22, 25, 38, 37, 9, 12, 34, 24, 42, 36,
	41, 26, 30, 21, 11, 8, 25, 17, 20, 19,
	35, 18, 6, 43, 39, 33, 4, 3, 1, 28,
	5, 31, 44, 45, 10, 2, 7,
}
var begatPact = [...]int{

	32, -1000, 32, -1000, 18, -1000, -1000, 10, -1000, 12,
	-1, -1000, 16, 17, 15, 14, 9, -5, 13, 13,
	-8, -8, -1000, -1000, -11, 31, -1000, -1000, -2, -1000,
	-1000, 3, 30, -16, -1000, -1000, -1000, 6, 4, -1000,
	29, -8, -8, -1000, -1000, -1000,
}
var begatPgo = [...]int{

	0, 37, 46, 45, 44, 41, 1, 0, 39, 38,
}
var begatR1 = [...]int{

	0, 9, 3, 3, 1, 2, 2, 4, 4, 4,
	4, 4, 5, 5, 5, 6, 6, 6, 6, 7,
	8, 8,
}
var begatR2 = [...]int{

	0, 1, 1, 2, 10, 3, 0, 0, 4, 4,
	4, 4, 0, 4, 4, 3, 1, 4, 2, 2,
	1, 2,
}
var begatChk = [...]int{

	-1000, -9, -3, -1, 4, -1, 14, -2, 15, 4,
	-4, 12, 16, 5, 6, 7, 8, 11, 14, 14,
	14, 14, 15, -6, 4, 13, -6, -7, -8, 17,
	-7, -5, 19, 4, 18, -6, 16, 10, 9, 4,
	19, 14, 14, 4, -7, -7,
}
var begatDef = [...]int{

	0, -2, 1, 2, 0, 3, 6, 0, 7, 0,
	0, 5, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 12, 8, 16, 0, 9, 10, 0, 20,
	11, 0, 0, 18, 19, 21, 4, 0, 0, 15,
	0, 0, 0, 17, 13, 14,
}
var begatTok1 = [...]int{

	1,
}
var begatTok2 = [...]int{

	2, 3, 4, 5, 6, 7, 8, 9, 10, 11,
	12, 13, 14, 15, 16, 17, 18, 19,
}
var begatTok3 = [...]int{
	0,
}

var begatErrorMessages = [...]struct {
	state int
	token int
	msg   string
}{}

//line yaccpar:1

/*	parser for yacc output	*/
// Code generated by goyacc. DO NOT EDIT.

var (
	begatDebug        = 0
	begatErrorVerbose = false
)

type begatLexer interface {
	Lex(lval *begatSymType) int
	Error(s string)
}

type begatParser interface {
	Parse(begatLexer) int
	Lookahead() int
}

type begatParserImpl struct {
	lval  begatSymType
	stack [begatInitialStackSize]begatSymType
	char  int
}

func (p *begatParserImpl) Lookahead() int {
	return p.char
}

func begatNewParser() begatParser {
	return &begatParserImpl{}
}

const begatFlag = -1000

func begatTokname(c int) string {
	if c >= 1 && c-1 < len(begatToknames) {
		if begatToknames[c-1] != "" {
			return begatToknames[c-1]
		}
	}
	return __yyfmt__.Sprintf("tok-%v", c)
}

func begatStatname(s int) string {
	if s >= 0 && s < len(begatStatenames) {
		if begatStatenames[s] != "" {
			return begatStatenames[s]
		}
	}
	return __yyfmt__.Sprintf("state-%v", s)
}

func begatErrorMessage(state, lookAhead int) string {
	const TOKSTART = 4

	if !begatErrorVerbose {
		return "syntax error"
	}

	for _, e := range begatErrorMessages {
		if e.state == state && e.token == lookAhead {
			return "syntax error: " + e.msg
		}
	}

	res := "syntax error: unexpected " + begatTokname(lookAhead)

	// To match Bison, suggest at most four expected tokens.
	expected := make([]int, 0, 4)

	// Look for shiftable tokens.
	base := begatPact[state]
	for tok := TOKSTART; tok-1 < len(begatToknames); tok++ {
		if n := base + tok; n >= 0 && n < begatLast && begatChk[begatAct[n]] == tok {
			if len(expected) == cap(expected) {
				return res
			}
			expected = append(expected, tok)
		}
	}

	if begatDef[state] == -2 {
		i := 0
		for begatExca[i] != -1 || begatExca[i+1] != state {
			i += 2
		}

		// Look for tokens that we accept or reduce.
		for i += 2; begatExca[i] >= 0; i += 2 {
			tok := begatExca[i]
			if tok < TOKSTART || begatExca[i+1] == 0 {
				continue
			}
			if len(expected) == cap(expected) {
				return res
			}
			expected = append(expected, tok)
		}

		// If the default action is to accept or reduce, give up.
		if begatExca[i+1] != 0 {
			return res
		}
	}

	for i, tok := range expected {
		if i == 0 {
			res += ", expecting "
		} else {
			res += " or "
		}
		res += begatTokname(tok)
	}
	return res
}

func begatlex1(lex begatLexer, lval *begatSymType) (char, token int) {
	token = 0
	char = lex.Lex(lval)
	if char <= 0 {
		token = begatTok1[0]
		goto out
	}
	if char < len(begatTok1) {
		token = begatTok1[char]
		goto out
	}
	if char >= begatPrivate {
		if char < begatPrivate+len(begatTok2) {
			token = begatTok2[char-begatPrivate]
			goto out
		}
	}
	for i := 0; i < len(begatTok3); i += 2 {
		token = begatTok3[i+0]
		if token == char {
			token = begatTok3[i+1]
			goto out
		}
	}

out:
	if token == 0 {
		token = begatTok2[1] /* unknown char */
	}
	if begatDebug >= 3 {
		__yyfmt__.Printf("lex %s(%d)\n", begatTokname(token), uint(char))
	}
	return char, token
}

func begatParse(begatlex begatLexer) int {
	return begatNewParser().Parse(begatlex)
}

func (begatrcvr *begatParserImpl) Parse(begatlex begatLexer) int {
	var begatn int
	var begatVAL begatSymType
	var begatDollar []begatSymType
	_ = begatDollar // silence set and not used
	begatS := begatrcvr.stack[:]

	Nerrs := 0   /* number of errors */
	Errflag := 0 /* error recovery flag */
	begatstate := 0
	begatrcvr.char = -1
	begattoken := -1 // begatrcvr.char translated into internal numbering
	defer func() {
		// Make sure we report no lookahead when not parsing.
		begatstate = -1
		begatrcvr.char = -1
		begattoken = -1
	}()
	begatp := -1
	goto begatstack

ret0:
	return 0

ret1:
	return 1

begatstack:
	/* put a state and value onto the stack */
	if begatDebug >= 4 {
		__yyfmt__.Printf("char %v in %v\n", begatTokname(begattoken), begatStatname(begatstate))
	}

	begatp++
	if begatp >= len(begatS) {
		nyys := make([]begatSymType, len(begatS)*2)
		copy(nyys, begatS)
		begatS = nyys
	}
	begatS[begatp] = begatVAL
	begatS[begatp].yys = begatstate

begatnewstate:
	begatn = begatPact[begatstate]
	if begatn <= begatFlag {
		goto begatdefault /* simple state */
	}
	if begatrcvr.char < 0 {
		begatrcvr.char, begattoken = begatlex1(begatlex, &begatrcvr.lval)
	}
	begatn += begattoken
	if begatn < 0 || begatn >= begatLast {
		goto begatdefault
	}
	begatn = begatAct[begatn]
	if begatChk[begatn] == begattoken { /* valid shift */
		begatrcvr.char = -1
		begattoken = -1
		begatVAL = begatrcvr.lval
		begatstate = begatn
		if Errflag > 0 {
			Errflag--
		}
		goto begatstack
	}

begatdefault:
	/* default state action */
	begatn = begatDef[begatstate]
	if begatn == -2 {
		if begatrcvr.char < 0 {
			begatrcvr.char, begattoken = begatlex1(begatlex, &begatrcvr.lval)
		}

		/* look through exception table */
		xi := 0
		for {
			if begatExca[xi+0] == -1 && begatExca[xi+1] == begatstate {
				break
			}
			xi += 2
		}
		for xi += 2; ; xi += 2 {
			begatn = begatExca[xi+0]
			if begatn < 0 || begatn == begattoken {
				break
			}
		}
		begatn = begatExca[xi+1]
		if begatn < 0 {
			goto ret0
		}
	}
	if begatn == 0 {
		/* error ... attempt to resume parsing */
		switch Errflag {
		case 0: /* brand new error */
			begatlex.Error(begatErrorMessage(begatstate, begattoken))
			Nerrs++
			if begatDebug >= 1 {
				__yyfmt__.Printf("%s", begatStatname(begatstate))
				__yyfmt__.Printf(" saw %s\n", begatTokname(begattoken))
			}
			fallthrough

		case 1, 2: /* incompletely recovered error ... try again */
			Errflag = 3

			/* find a state where "error" is a legal shift action */
			for begatp >= 0 {
				begatn = begatPact[begatS[begatp].yys] + begatErrCode
				if begatn >= 0 && begatn < begatLast {
					begatstate = begatAct[begatn] /* simulate a shift of "error" */
					if begatChk[begatstate] == begatErrCode {
						goto begatstack
					}
				}

				/* the current p has no shift on "error", pop stack */
				if begatDebug >= 2 {
					__yyfmt__.Printf("error recovery pops state %d\n", begatS[begatp].yys)
				}
				begatp--
			}
			/* there is no state on the stack with an error shift ... abort */
			goto ret1

		case 3: /* no shift yet; clobber input char */
			if begatDebug >= 2 {
				__yyfmt__.Printf("error recovery discards %s\n", begatTokname(begattoken))
			}
			if begattoken == begatEofCode {
				goto ret1
			}
			begatrcvr.char = -1
			begattoken = -1
			goto begatnewstate /* try again in the same state */
		}
	}

	/* reduction by production begatn */
	if begatDebug >= 2 {
		__yyfmt__.Printf("reduce %v in:\n\t%v\n", begatn, begatStatname(begatstate))
	}

	begatnt := begatn
	begatpt := begatp
	_ = begatpt // guard against "declared and not used"

	begatp -= begatR2[begatn]
	// begatp is now the index of $0. Perform the default action. Iff the
	// reduced production is ε, $1 is possibly out of range.
	if begatp+1 >= len(begatS) {
		nyys := make([]begatSymType, len(begatS)*2)
		copy(nyys, begatS)
		begatS = nyys
	}
	begatVAL = begatS[begatp+1]

	/* consult goto table to find next state */
	begatn = begatR1[begatn]
	begatg := begatPgo[begatn]
	begatj := begatg + begatS[begatp].yys + 1

	if begatj >= begatLast {
		begatstate = begatAct[begatg]
	} else {
		begatstate = begatAct[begatj]
		if begatChk[begatstate] != -begatn {
			begatstate = begatAct[begatg]
		}
	}
	// dummy call; replaced with literal code
	switch begatnt {

	case 1:
		begatDollar = begatS[begatpt-1 : begatpt+1]
		//line parse.y:45
		{
			begatTests = begatDollar[1].btests
		}
	case 2:
		begatDollar = begatS[begatpt-1 : begatpt+1]
		//line parse.y:49
		{
			begatVAL.btests = make([]*begatTest, 1)
			begatVAL.btests[0] = begatDollar[1].btest
		}
	case 3:
		begatDollar = begatS[begatpt-2 : begatpt+1]
		//line parse.y:53
		{
			begatVAL.btests = append(begatDollar[1].btests, begatDollar[2].btest)
		}
	case 4:
		begatDollar = begatS[begatpt-10 : begatpt+1]
		//line parse.y:57
		{
			begatVAL.btest = &begatTest{pre: begatDollar[5].pre, post: begatDollar[9].post}
			begatVAL.btest.prior = begatDollar[3].btest.prior
			begatVAL.btest.name = begatDollar[1].str
		}
	case 5:
		begatDollar = begatS[begatpt-3 : begatpt+1]
		//line parse.y:63
		{
			begatVAL.btest = begatDollar[1].btest
			begatVAL.btest.prior = append(begatVAL.btest.prior, begatDollar[2].str)
		}
	case 6:
		begatDollar = begatS[begatpt-0 : begatpt+1]
		//line parse.y:67
		{
			begatVAL.btest = &begatTest{}
		}
	case 7:
		begatDollar = begatS[begatpt-0 : begatpt+1]
		//line parse.y:71
		{
			begatVAL.pre = begatPre{}
		}
	case 8:
		begatDollar = begatS[begatpt-4 : begatpt+1]
		//line parse.y:74
		{
			begatVAL.pre = begatDollar[1].pre
			begatVAL.pre.begatfile = begatDollar[4].thing
		}
	case 9:
		begatDollar = begatS[begatpt-4 : begatpt+1]
		//line parse.y:78
		{
			begatVAL.pre = begatDollar[1].pre
			begatVAL.pre.history = begatDollar[4].thing
		}
	case 10:
		begatDollar = begatS[begatpt-4 : begatpt+1]
		//line parse.y:82
		{
			begatVAL.pre = begatDollar[1].pre
			begatVAL.pre.pastiche = begatDollar[4].thingset
		}
	case 11:
		begatDollar = begatS[begatpt-4 : begatpt+1]
		//line parse.y:86
		{
			begatVAL.pre = begatDollar[1].pre
			begatVAL.pre.inputs = begatDollar[4].thingset
		}
	case 12:
		begatDollar = begatS[begatpt-0 : begatpt+1]
		//line parse.y:91
		{
			begatVAL.post = begatPost{}
		}
	case 13:
		begatDollar = begatS[begatpt-4 : begatpt+1]
		//line parse.y:94
		{
			begatVAL.post = begatDollar[1].post
			begatVAL.post.dictums = begatDollar[4].thingset
		}
	case 14:
		begatDollar = begatS[begatpt-4 : begatpt+1]
		//line parse.y:98
		{
			begatVAL.post = begatDollar[1].post
			begatVAL.post.outputs = begatDollar[4].thingset
		}
	case 15:
		begatDollar = begatS[begatpt-3 : begatpt+1]
		//line parse.y:103
		{
			begatVAL.thing = begatThing{add: true, id: begatDollar[1].str, chk: begatDollar[3].str}
		}
	case 16:
		begatDollar = begatS[begatpt-1 : begatpt+1]
		//line parse.y:106
		{
			begatVAL.thing = begatThing{add: true, id: begatDollar[1].str}
		}
	case 17:
		begatDollar = begatS[begatpt-4 : begatpt+1]
		//line parse.y:109
		{
			begatVAL.thing = begatThing{add: false, id: begatDollar[1].str, chk: begatDollar[3].str}
		}
	case 18:
		begatDollar = begatS[begatpt-2 : begatpt+1]
		//line parse.y:112
		{
			begatVAL.thing = begatThing{add: false, id: begatDollar[1].str}
		}
	case 19:
		begatDollar = begatS[begatpt-2 : begatpt+1]
		//line parse.y:116
		{
			begatVAL.thingset = begatDollar[1].thingset
		}
	case 20:
		begatDollar = begatS[begatpt-1 : begatpt+1]
		//line parse.y:120
		{
			begatVAL.thingset = make([]begatThing, 0)
		}
	case 21:
		begatDollar = begatS[begatpt-2 : begatpt+1]
		//line parse.y:123
		{
			begatVAL.thingset = append(begatDollar[1].thingset, begatDollar[2].thing)
		}
	}
	goto begatstack /* stack new state and value */
}
