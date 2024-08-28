// This is the yacc input for creating the begat testgen parser
//yacc:flags -p begat

%{
package cmd

import (
	"io"
		"fmt"
)

var begatTests []*begatTest

func init() {
	begatErrorVerbose = true
}

%}

%union {
	btests	[]*begatTest
	btest	*begatTest
	str	string
	pre	begatPre
	post	begatPost
	thing	begatThing
	thingset	[]begatThing
}

%type	<btest>		test priors
%type	<btests>	tests
%type	<pre>		pre
%type	<post>		post
%type	<thing>		thing
%type	<thingset>	thingset thingset1

%token  <str>	IDENTIFIER
%token  <str>	BEGATFILE HISTORY PASTICHE INPUTS OUTPUTS DICTUMS
%token	<str>	ARROW PLUS MINUS COLON LBRACE RBRACE LBRACK RBRACK EQUAL

//%left PLUS MINUS

%%

everything:	tests {
		begatTests = $1
}

tests:	test {
		$$ = make([]*begatTest, 1)
		$$[0] = $1
	}
|	tests test {
		$$ = append($1, $2)
	}

test:	IDENTIFIER COLON priors LBRACE pre RBRACE ARROW LBRACE post RBRACE {
		$$ = &begatTest{ pre:$5, post:$9 }
		$$.prior = $3.prior
		$$.name = $1
	}

priors:	priors IDENTIFIER PLUS {
		$$ = $1
		$$.prior = append($$.prior, $2)
	}
|	{
		$$ = &begatTest{ }
	}

pre:	{
		$$ = begatPre{}
	}
|	pre BEGATFILE COLON thing {
		$$ = $1
		$$.begatfile = $4
	}
|	pre HISTORY COLON thing {
		$$ = $1
		$$.history = $4
	}
|	pre PASTICHE COLON thingset {
		$$ = $1
		$$.pastiche = $4
	}
|	pre INPUTS COLON thingset {
		$$ = $1
		$$.inputs = $4
	}

post:	{
		$$ = begatPost{}
	}
|	post DICTUMS COLON thingset {
		$$ = $1
		$$.dictums = $4
	}
|	post OUTPUTS COLON thingset {
		$$ = $1
		$$.outputs = $4
	}

thing:	IDENTIFIER EQUAL IDENTIFIER {
		$$ = begatThing{ add:true, id:$1, chk:$3}
	}
|	IDENTIFIER {
		$$ = begatThing{add:true,id:$1}
	}
|	MINUS IDENTIFIER EQUAL IDENTIFIER {
		$$ = begatThing{ add:false, id:$1, chk:$3}
	}
|	MINUS IDENTIFIER {
		$$ = begatThing{add:false,id:$1}
	}

thingset:	thingset1 RBRACK {
		$$ = $1
	}

thingset1:	LBRACK {
		$$ = make([]begatThing, 0)
	}
|	thingset1 thing {
		$$ = append($1, $2)
	}

%%

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
