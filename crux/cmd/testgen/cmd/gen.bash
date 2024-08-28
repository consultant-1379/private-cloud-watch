# repair output of yacc and nex so that it passes vet and lint
golex lexer.l
awk '
NR==1 { printf("// Package cmd is a package.\n") }
/var EOF/ { printf("// EOF is used to denote end-of-file\n") }
/var INITIAL/ { printf("// INITIAL is used to affect start of parsing\n") }
/var YY_START/ { printf("// YY_START needs to be defined\n") }
{print $0}' < lexer.l.go | sed 's/ch_stop/chStop/g
s/yyorigidx += 1/yyorigidx++/g
s/yyRT_FALLTHROUGH/yyRTFALLTHROUGH/g
s/yyRT_USER_RETURN/yyRTUSERRETURN/g
s/yyRT_REJECT/yyRTREJECT/g
s/YY_START/YYSTART/g
s/var yydata string = ""/var yydata string/
s/var yytext string = ""/var yytext string/
s/var yytextrepl bool = true/var yytextrepl = true/
s/var EOF int = -1/var EOF = -1/
s/var INITIAL yystartcondition = 0/var INITIAL yystartcondition/
s/var YYSTART yystartcondition = INITIAL/var YYSTART = INITIAL/
s/var yyrules \[\]yyrule/var yyrules/
s/begatLex/begatlex/g' > lexer.go; rm lexer.l.go

goyacc -p begat -o crap parse.y
awk '/^const/ { printf("// %s is a token\n", $2)}
{print $0}' < crap | sed 's/begatEofCode/begatEOFCode/g
s/parsercvr/p/g
/^func.*Lookahead/s/p/begatrcvr/
s/p\.char/begatrcvr.char/' > parse.go; rm crap
