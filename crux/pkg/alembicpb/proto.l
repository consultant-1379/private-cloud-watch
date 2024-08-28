%{
package alembicpbgen

import (
	"strconv"
)

var lval ProtoSymType
var yylineno int = 1
var yyfile string = "stdin"
%}

%%

\n		{ yylineno++ }
returns		{ return RETURNS }
enum		{ return ENUM }
rpc		{ return RPC }
syntax		{ return SYNTAX }
stream		{ return STREAM }
service		{ return SERVICE }
package		{ return PACKAGE }
message		{ return MESSAGE }
repeated	{ return REPEATED }
import		{ return IMPORT }
";"		{ return SEMICOLON }
,		{ return COMMA }
"("		{ return LPAR }
")"		{ return RPAR }
=		{ return EQUAL}
"{"		{ return LBRACE }
"}"		{ return RBRACE }
syntax		{ return SYNTAX }
=		{ return EQUAL }
\"[^"]*\"		{ x := yytext; lval.str = x[1:len(x)-1]; return STRING }
[_A-Za-z][_A-Za-z0-9]*	{ lval.str = yytext; return NAME }
[0-9]+		{ lval.num,_ = strconv.Atoi(yytext); return NUMBER }
\/\/PRE\ .*	{ x := yytext; lval.str = x[6:]; return PRE }
\/\/.*		{ }
[ \t]		{ }

%%
