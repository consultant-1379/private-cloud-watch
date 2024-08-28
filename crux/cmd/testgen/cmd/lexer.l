%{
package cmd

var lval begatSymType
var yylineno = 1
var yyfile = "stdin"
%}

%%

\n		{ yylineno++ }
begatfile	{ return BEGATFILE }
history		{ return HISTORY }
pastiche	{ return PASTICHE }
inputs		{ return INPUTS }
outputs		{ return OUTPUTS }
dictums		{ return DICTUMS }
=>		{ return ARROW }
\+		{ return PLUS }
-		{ return MINUS }
:		{ return COLON }
\{		{ return LBRACE }
\}		{ return RBRACE }
\[		{ return LBRACK }
\]		{ return RBRACK }
=		{ return EQUAL }
[a-zA-Z0-9_.\/]* { lval.str = yytext; return IDENTIFIER }
#.*		{ }
\/\/.*		{ }
[ \t]		{ }

%%
