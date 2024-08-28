// This is the yacc input for creating the khan parser
//yacc:flags -p Parse

%{
package khangen

import (
	"strconv"
	"io"
	"fmt"
	kd "github.com/erixzone/crux/pkg/khan/defn"
)

var ParsedExprs []*kd.Expr
var ParsedAfters []*kd.After

%}

%union {
	num		int
	str		string
	expr		*kd.Expr
	exprlist	[]*kd.Expr
	after		*kd.After
	afterlist	[]*kd.After
	condition	*kd.Condition
	conditionlist	[]*kd.Condition
	pair		[]string
	truth		bool
	strlist		[]string
}

%type	<exprlist>	spec
%type	<expr>		expr number
%type	<after>		after
%type	<afterlist>	afters
%type	<pair>		state
%type	<truth>		percent
%type	<condition>	cond
%type	<conditionlist>	conds
%type	<strlist>	slist

%token  <str> NUMBER STRING
%token  <str> IDENTIFIER
%token  <str> PLUS MINUS MULT DIVIDE COMMA AND NOT OR ASSIGNS LPAR RPAR
%token  <str> PICK PICKH SIZE LABEL ALL
%token	<str> START AFTER OF PERCENT DOT

%right ASSIGNS
%right NOT
%left OR
%left AND
%left PLUS MINUS
%left MULT DIVIDE

%%

program:	spec afters {
		ParsedExprs = $1
		ParsedAfters = $2
}
spec:	expr {
		$$ = append(make([]*kd.Expr, 0), $1)
	}
|	spec expr {
		$$ = append($1, $2)
	}

expr : IDENTIFIER ASSIGNS expr {
		$$ = &kd.Expr{Op:kd.OpAssign, Str:$1, ExprL:$3}
		if false { fmt.Printf("OMG %s := %v\n", $1, $3)}
	}
|	IDENTIFIER ASSIGNS LPAR slist RPAR {
       		$$ = &kd.Expr{Op:kd.OpVAssign, Str:$1, Strlist:$4}
       	}
|	PICK LPAR IDENTIFIER  COMMA number COMMA expr RPAR	{
		$$ = &kd.Expr{Op:kd.OpPick, Str:$3, ExprL:$5, ExprR:$7}
	}
|	PICKH LPAR IDENTIFIER  COMMA number COMMA expr RPAR	{
		$$ = &kd.Expr{Op:kd.OpPickh, Str:$3, ExprL:$5, ExprR:$7}
	}
|	ALL {
		$$ = &kd.Expr{Op:kd.OpAll}
	}
|	LABEL LPAR IDENTIFIER RPAR {
		$$ = &kd.Expr{Op:kd.OpLabel, Str:$3}
	}
|	NOT expr {
		// complement (w.r.t ALL)
		$$ = &kd.Expr{Op:kd.OpNot, ExprL:$2}
	}
|	expr AND expr {
		$$ = &kd.Expr{Op:kd.OpAnd, ExprL:$1, ExprR:$3}
	}
|	expr OR expr {
		$$ = &kd.Expr{Op:kd.OpOr, ExprL:$1, ExprR:$3}
	}
|	IDENTIFIER {
		$$ = &kd.Expr{Op:kd.OpVar, Str:$1 }
	}

slist :	STRING {
		$$ = make([]string, 1)
		$$[0] = $1
	}
|	slist COMMA STRING {
		$$ = append($1, $3)
	}
afters :		{
		$$ = make([]*kd.After, 0)
	}
|	afters after {
		$$ = append($1, $2)
	}

after :  START state AFTER conds {
		$$ = &kd.After{ Service:$2[0], Stage:$2[1], Pre:$4}
	}
	
conds :	cond	{
		$$ = append(make([]*kd.Condition, 0), $1)
	}
|	conds cond {
		$$ = append($1, $2)
	}

cond :	state {
		$$ = &kd.Condition{IsCount:false, Vale:&kd.Expr{Op:kd.OpNum, Num:100.0}, Service:$1[0], Stage:$1[1]}
	}
|	number percent OF state	{
		$$ = &kd.Condition{IsCount:!$2, Vale:$1, Service:$4[0], Stage:$4[1]}
	}

state	:	IDENTIFIER {
		$$ = make([]string, 2)
		$$[0] = $1
	}
|	IDENTIFIER DOT IDENTIFIER {
		$$ = make([]string, 2)
		$$[0] = $1
		$$[1] = $3
	}

percent	:	{ $$ = false }
|	PERCENT	{ $$ = true }

number :	NUMBER {
		val,_ := strconv.ParseFloat($1, 64)
		$$ = &kd.Expr{Op:kd.OpNum, Num:val}
	}
|	SIZE LPAR expr RPAR {
		$$ = &kd.Expr{Op:kd.OpSize, ExprL:$3}
	}
|	number PLUS number {
		$$ = &kd.Expr{Op:kd.OpPlus, ExprL:$1, ExprR:$3}
	}
|	number MINUS number {
		$$ = &kd.Expr{Op:kd.OpMinus, ExprL:$1, ExprR:$3}
	}
|	number MULT number {
		$$ = &kd.Expr{Op:kd.OpMult, ExprL:$1, ExprR:$3}
	}
|	number DIVIDE number {
		$$ = &kd.Expr{Op:kd.OpDivide, ExprL:$1, ExprR:$3}
	}


%%

type FlexLex struct {
	lval *ParseSymType
}

func (fl *FlexLex) Lex(lv *ParseSymType) int {
	ret := yylex()
	*lv = lval
	return ret
}

func (fl *FlexLex) Error(e string) {
	fmt.Printf("error:%s:%d: %s\n", yyfile, yylineno, e)
}

func Parse(filename string, rdr io.Reader, noisy bool) error {
	if noisy {
		ParseDebug = 4
	} else {
		ParseDebug = 0
	}
	var fl FlexLex
	yyin = rdr
	yyfile = filename
	ret := ParseParse(&fl)
	if ret == 1 {
		return fmt.Errorf("parse failed")
	}
	return nil
}
