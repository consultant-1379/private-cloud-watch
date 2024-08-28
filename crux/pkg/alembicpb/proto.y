// Parse the relevent bit of a protobuf spec
//yacc:flags -p Proto

%{
package alembicpbgen

import (
	"fmt"
	"io"

	"github.com/erixzone/crux/pkg/alembicpb"
)

var Services []*alembicpb.ProtoService
var Pre string
var Package string

%}

%union {
	line		int
	column		int
	num		int
	str		string
	service		*alembicpb.ProtoService
	traffic		alembicpb.ProtoTraffic
}

%type	<service>	service shead
%type	<traffic>	traffic

%token  <num> NUMBER
%token  <str> NAME STRING PRE
%token	<str> RETURNS RPC STREAM SERVICE PACKAGE MESSAGE IMPORT ENUM
%token  <str> COMMA LPAR RPAR LBRACE RBRACE
%token	<str> SYNTAX EQUAL SEMICOLON REPEATED

%%

defns:		SYNTAX EQUAL STRING SEMICOLON
|		defns PRE	{ Pre = $2 }
|		defns PACKAGE NAME SEMICOLON	{ Package = $3 }
|		defns IMPORT STRING SEMICOLON
|		defns enum
|		defns message
|		defns service

enum:		ENUM NAME LBRACE elist RBRACE

elist:		NAME EQUAL NUMBER SEMICOLON
|		elist NAME EQUAL NUMBER SEMICOLON

message:	mhead RBRACE

mhead:		MESSAGE NAME LBRACE
|		mhead NAME mname EQUAL NUMBER SEMICOLON
|		mhead REPEATED NAME mname EQUAL NUMBER SEMICOLON
|		mhead message

mname:		NAME
|		REPEATED
|		PACKAGE
|		MESSAGE

service:	shead RBRACE	{ Services = append(Services, $1) }

shead:		SERVICE NAME LBRACE					{ $$ = &alembicpb.ProtoService{Name:$2}; }
|		shead RPC NAME traffic RETURNS traffic LBRACE RBRACE	{ $$ = $1; $$.RPC = append($$.RPC, alembicpb.ProtoRPC{Name:$3, Req:$4, Reply:$6}); }

traffic:	LPAR NAME RPAR		{ $$ = alembicpb.ProtoTraffic{Name:$2, Streaming:false} }
|		LPAR STREAM NAME RPAR	{ $$ = alembicpb.ProtoTraffic{Name:$3, Streaming:true} }

%%

type FlexLex struct {
	lval *ProtoSymType
}

func (fl *FlexLex) Lex(lv *ProtoSymType) int {
	ret := yylex()
	*lv = lval
	return ret
}

func (fl *FlexLex) Error(e string) {
	fmt.Printf("error:%s:%d: %s\n", yyfile, yylineno, e)
}

func Parse(filename string, rdr io.Reader, noisy bool) (*alembicpb.ProtoPkg, error) {
	ProtoErrorVerbose = true
	if noisy {
		ProtoDebug = 4
	} else {
		ProtoDebug = 0
	}
	Services = make([]*alembicpb.ProtoService, 0)
	// nex creates the function: func NewLexerWithInit(in io.Reader, initFun func(*Lexer)) *Lexer
	// go tool yacc creates the function: func yyParse(yylex yyLexer) int
	var fl FlexLex
	yyin = rdr
	yyfile = filename
	ret := ProtoParse(&fl)
	if ret == 1 {
		return nil, fmt.Errorf("parse failed")
	}
	return &alembicpb.ProtoPkg{Services:Services, Pre:Pre, Package:Package}, nil
}
