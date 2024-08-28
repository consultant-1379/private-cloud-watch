package lib

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"unicode"
	"unicode/utf8"
)

// LexDebug should be replaced by finegrained debugging
var LexDebug = false

const (
	runeEOF = '\001' // need a better answer than this
)

// Token is the lexical unit
type Token struct {
	what tokenType
	file string
	line int
	val  []byte
}

func (t Token) String() string {
	return fmt.Sprintf("[%s:%d] %s '%s'", t.file, t.line, t.what, t.val)
}

// Lexer is a struct containing a linked list of lexing machines
type lexer struct {
	lex    *lexMach   // a stack of lexing machines
	tokens chan Token // channel of scanned tokens
}

// Lex1 is a lexing machine for a single input
type lexMach struct {
	file    string   // where input is coming from (most ften a pathname)
	lineNum int      // current line number within input (first line is 1)
	input   []byte   // all input
	start   int      // start position of last token
	pos     int      // current position of lexer in input
	width   int      // width of last token
	state   lexState // current state
	chain   *lexMach // where we were (in case we stacked inputs , e.g. via <)
}

// lexState embodies the current lexical state as
// a function that returns the next state
type lexState func(*lexer) lexState

// generate a new lexer
func newLexer(src string, rdr io.Reader) *lexer {
	lex := &lexer{nil, make(chan Token, 2)} // we just need two tokens
	lex.push(src, rdr)
	go lex.run() // concurrently run state machine
	return lex
}

// top-level lex a file
func lexFile(pathname string) *lexer {
	file, err := os.Open(pathname)
	if err != nil {
		panic(err)
	}
	return newLexer(pathname, bufio.NewReader(file))
}

// process a lexing machine until its done
func (lex *lexer) run() {
	for state := lex.lex.state; state != nil; {
		state = state(lex)
		if state == nil {
			// EOF; pop lexing machine
			state = lex.pop()
		}
	}
	// genuine EOF; send an eof and close
	lex.emit(tokenEOF)
	close(lex.tokens)
}

// push a new input onto the lexing stack
func (lex *lexer) push(src string, rdr io.Reader) {
	input := make([]byte, 0)
	buf := make([]byte, 4096) // the number doesn't matter much
	for {
		n, err := rdr.Read(buf)
		if err == io.EOF {
			break
		}
		input = append(input, buf[0:n]...)
	}
	lex.lex = &lexMach{file: src, lineNum: 1, start: 0, pos: 0, width: 0,
		input: input, state: lexStateNormal, chain: lex.lex}
}

// pop an input off the lexing stack
func (lex *lexer) pop() lexState {
	lex.lex = lex.lex.chain
	if lex.lex == nil {
		return nil
	}
	return lexStateNormal
}

// emit sends a Token back
func (lex *lexer) emit(t tokenType) {
	var tok Token
	if lex.lex != nil {
		tok = Token{what: t, file: lex.lex.file, line: lex.lex.lineNum, val: lex.lex.input[lex.lex.start:lex.lex.pos]}
		lex.lex.start = lex.lex.pos
	} else {
		tok = Token{what: t, line: 0, val: []byte{}}
	}
	lex.tokens <- tok
	if LexDebug {
		fmt.Printf("LEX: %s\n", tok)
	}
}

// stringify a lexMach
func (l *lexMach) String() string {
	if l == nil {
		return fmt.Sprintf("<nil lexMach>")
	}
	return fmt.Sprintf("%s:%d: start=%d pos=%d", l.file, l.lineNum, l.start, l.pos)
}

// ignore skips over all the input before the cursor
func (l *lexMach) ignore() {
	l.start = l.pos
}

// backup steps back input by the last rune returned by next
func (l *lexMach) backup() {
	l.pos -= l.width
}

// peek returns the next rune but does not consume it
func (l *lexMach) peek() rune {
	r, _ := l.next()
	l.backup()
	return r
}

// next returns the next rune
func (l *lexMach) next() (r rune, err error) {
	err = nil
	if l.pos >= len(l.input) {
		r = runeEOF
		return r, nil
	}
	r, l.width = utf8.DecodeRune(l.input[l.pos:])
	l.pos += l.width
	return r, err
}

// errorfn emits an error Token and shuts down processing by returning a nil state
func (lex *lexer) errorfn(file string, line int, format string, args ...interface{}) lexState {
	lex.tokens <- Token{what: tokenError, file: file, line: line, val: []byte(fmt.Sprintf(format, args...))}
	lex.lex.chain = nil // stop processing any more (stacked) input
	return nil
}

// is this rune part of a word? this will extend over time, i'm sure
func isWord(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || strings.ContainsRune("_/.%$", r)
}

// the various state machines for lexing
func lexStateNormal(lex *lexer) lexState {
	l := lex.lex
loop:
	for {
		r, err := l.next()
		if err != nil {
			return lex.errorfn(l.file, l.lineNum, "%s.%d: %s", l.file, l.lineNum, err)
		}
		switch {
		case r == runeEOF:
			return nil
		case r == '\n':
			l.ignore()
			lex.emit(tokenNL)
			l.lineNum++
		case r == '#':
			for {
				r, err = l.next()
				if err != nil {
					return lex.errorfn(l.file, l.lineNum, "%s.%d: %s", l.file, l.lineNum, err)
				}
				if (r == runeEOF) || (r == '\n') {
					continue loop
				}
			}
		case r == ':':
			l.ignore()
			lex.emit(tokenColon)
		case r == '=':
			l.ignore()
			lex.emit(tokenEqual)
		case r == '[':
			l.ignore()
			if l.peek() == '[' {
				l.next()
				l.ignore()
				if l.peek() == '[' {
					l.next()
					l.ignore()
					return lexStateVerbatim2
				}
				if l.peek() == '\n' {
					l.next()
					l.ignore()
				}
				return lexStateVerbatim
			}
			lex.emit(tokenLSB)
		case r == ']':
			l.ignore()
			lex.emit(tokenRSB)
		case r == '{':
			l.ignore()
			lex.emit(tokenLbrace)
		case r == '}':
			l.ignore()
			lex.emit(tokenRbrace)
		case r == '(':
			l.ignore()
			lex.emit(tokenLpar)
		case r == ')':
			l.ignore()
			lex.emit(tokenRpar)
		case r == '"':
			l.ignore()
			return lexStateString
		case unicode.IsSpace(r):
			l.ignore()
			lex.emit(tokenWS)
		case r == '\\':
			r = l.peek()
			if r == '\n' {
				l.ignore()
				l.next()
				l.ignore()
				lex.emit(tokenWS)
				l.lineNum++
			}
		case r == '<':
			r = l.peek()
			if r == '|' {
				l.next()
				return lexStatePipe
			}
			return lexStateFile
		case isWord(r):
			// i'm unhappy about this; there shouldn't be a special case for $
			if r == '$' {
				r, _ = l.next()
				if r == '{' {
					for r != runeEOF {
						r, _ = l.next()
						if r == '}' {
							l.next()
							break
						}
					}
				} else {
					for isWord(r) {
						r, _ = l.next()
					}
				}
			} else {
				for isWord(r) {
					r, _ = l.next()
				}
			}
			l.backup()
			lex.emit(tokenWord)
		default:
			lex.emit(tokenOther)
		}
	}
}

// process a string
func lexStateString(lex *lexer) lexState {
	l := lex.lex
	for {
		r, err := l.next()
		if err != nil {
			return lex.errorfn(l.file, l.lineNum, "%s.%d: %s", l.file, l.lineNum, err)
		}
		switch {
		case r == runeEOF:
			return lex.errorfn(l.file, l.lineNum, "%s.%d: %s", l.file, l.lineNum, "unexpected EOF")
		case r == '\n':
			return lex.errorfn(l.file, l.lineNum, "%s.%d: %s", l.file, l.lineNum, "unexpected newline")
		case r == '"':
			l.backup() // undo "
			lex.emit(tokenString)
			_, _ = l.next() // re-eat "
			return lexStateNormal
		case r == '\\':
			r, err := l.next()
			if err != nil {
				return lex.errorfn(l.file, l.lineNum, "%s.%d: %s", l.file, l.lineNum, err)
			}
			if r == runeEOF {
				return lex.errorfn(l.file, l.lineNum, "%s.%d: %s", l.file, l.lineNum, "unexpected EOF")
			}
		default:

		}
	}
}

// process a verbatim (recipe) section
// we have absorbed(ignored) the initial [[
func lexStateVerbatim(lex *lexer) lexState {
	l := lex.lex
	lineBeg := false
	for {
		r, err := l.next()
		if err != nil {
			return lex.errorfn(l.file, l.lineNum, "%s.%d: %s", l.file, l.lineNum, err)
		}
		switch {
		case r == runeEOF:
			return lex.errorfn(l.file, l.lineNum, "%s.%d: %s", l.file, l.lineNum, "unexpected EOF")
		case r == '\n':
			lineBeg = true
		case lineBeg && (r == ']'):
			if l.peek() == ']' {
				l.pos -= 2 // ugh; there must be a better way to do this..
				lex.emit(tokenVerbatim)
				l.pos += 2      // undo above
				_, _ = l.next() // re-eat "
				return lexStateNormal
			}
			lineBeg = false
		default:
			lineBeg = false
		}
	}
}

// process a [[[ (inline) verbatim (recipe) section
// we have absorbed(ignored) the initial [[[]
func lexStateVerbatim2(lex *lexer) lexState {
	l := lex.lex
	for {
		r, err := l.next()
		if err != nil {
			return lex.errorfn(l.file, l.lineNum, "%s.%d: %s", l.file, l.lineNum, err)
		}
		switch {
		case r == runeEOF:
			return lex.errorfn(l.file, l.lineNum, "%s.%d: %s", l.file, l.lineNum, "unexpected EOF")
		case r == ']':
			if l.peek() == ']' {
				l.next()
				if l.peek() == ']' {
					l.pos = l.pos - 2 // ugh; there must be a better way to do this..
					lex.emit(tokenVerbatim)
					l.pos = l.pos + 2 // undo above
					l.next()
					return lexStateNormal
				}
			}
		}
	}
}

// process redirecting in from a file
func lexStateFile(lex *lexer) lexState {
	l := lex.lex
	for r, _ := l.next(); (r == ' ') || (r == '\t'); {
		r, _ = l.next()
	}
	l.backup()
	// l.pos is at first non-blank
	var beg int
	for beg = l.pos; (l.input[l.pos] != '\n') && (l.pos < len(l.input)); {
		l.pos++
	}
	filename := string(l.input[beg:l.pos])
	l.pos++
	l.lineNum++
	l.ignore()
	file, err := os.Open(filename)
	if err != nil {
		return lex.errorfn(l.file, l.lineNum, "%s\n", err)
	}
	lex.push(filename, file)
	return lexStateNormal
}

// process redirecting in from a pipe
func lexStatePipe(lex *lexer) lexState {
	l := lex.lex
	for r, _ := l.next(); (r == ' ') || (r == '\t'); {
		r, _ = l.next()
	}
	l.backup()
	// l.pos is at first non-blank
	var beg int
	for beg = l.pos; (l.input[l.pos] != '\n') && (l.pos < len(l.input)); {
		l.pos++
	}
	filename := string(l.input[beg:l.pos])
	l.pos++
	l.ignore()
	l.lineNum++
	cmd := exec.Command("/bin/sh", "-c", filename)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return lex.errorfn(l.file, l.lineNum, "%s\n", err)
	}
	go func() {
		err := cmd.Run()
		if err != nil {
			panic(err)
		}
	}()
	lex.push(filename, stdout)
	return lexStateNormal
}
