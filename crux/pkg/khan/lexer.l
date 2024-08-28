%{
package khangen

var lval ParseSymType
var yylineno int = 1
var yyfile string = "stdin"
%}

%%

\n		{ yylineno++ }
ALL		{ return ALL }
pick		{ return PICK }
pickh		{ return PICKH }
SIZE		{ return SIZE }
LABEL		{ return LABEL }
start		{ return START }
after		{ return AFTER }
of		{ return OF }
\+		{ return PLUS }
-		{ return MINUS }
:=		{ return ASSIGNS }
\*		{ return MULT  }
\/		{ return DIVIDE  }
\(		{ return LPAR  }
\)		{ return RPAR }
\,		{ return COMMA }
\&		{ return AND }
\!		{ return NOT }
\|		{ return OR }
\%		{ return PERCENT }
\.		{ return DOT }
\"[^"]*\"	{ lval.str = yytext[1:len(yytext)-1]; return STRING }
[0-9]+		{ lval.str = yytext; return NUMBER }
[a-zA-Z][a-zA-Z0-9_]*	{ lval.str = yytext; return IDENTIFIER }
[ \t]		{ }

%%
