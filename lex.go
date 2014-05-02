package rfc5424

import (
	"fmt"
	"strings"
        "unicode"
	"unicode/utf8"
)

type Pos int
type itemType int

const eof = -1

type item struct {
	typ itemType // The type of this item.
	pos Pos      // The starting position, in bytes, of this item in the input string.
	val string   // The value of this item.
}

func (i item) String() string {
	switch {
	case i.typ == itemEOF:
		return "EOF"
	case i.typ == itemError:
		return i.val
	case len(i.val) > 10:
		return fmt.Sprintf("%.10q...", i.val)
	}
	return fmt.Sprintf("%q", i.val)
}

// <86>1 2014-01-20T13:26:16-08:00 hostname appname pid msgid - message
const (
	itemError itemType = iota // error occurred
	itemEOF
	itemLeftAngleBracket
	itemRightAngleBracket
	itemNumber
	itemSpace
	itemHyphen
	itemLeftBracket
	itemRightBracket
	itemAt
	itemEqualSign
	itemString // quoted string (includes quotes)
	itemText   // plain text
	itemNil
)

type stateFn func(*lexer) stateFn

type lexer struct {
	name         string    // the name of the input; used only for error reports
	input        string    // the string being scanned
        leftAngle    string    // start of PRIVAL
        rightAngle   string    // end of PRIVAL
	leftBracket  string    // start of STRUCTURED-DATA
	rightBracket string    // end of STRUCTURED-DATA
        state        stateFn   // the next lexing function to enter
	pos          Pos       // current position in the input
	start        Pos       // start position of this item
        width        Pos       // width of last rune read from input
        lastPos      Pos       // position of most recent item returned by nextItem
	items        chan item // channel of scanned items
}

func (l *lexer) next() rune {
	if int(l.pos) >= len(l.input) {
		l.width = 0
		return eof
	}
	r, w := utf8.DecodeRuneInString(l.input[l.pos:])
	l.width = Pos(w)
	l.pos += l.width
	return r
}

func (l *lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

func (l *lexer) backup() {
	l.pos -= l.width
}

func (l *lexer) emit(t itemType) {
	l.items <- item{t, l.start, l.input[l.start:l.pos]}
	l.start = l.pos
}

func (l *lexer) ignore() {
	l.start = l.pos
}

func (l *lexer) accept(valid string) bool {
	if strings.IndexRune(valid, l.next()) >= 0 {
		return true
	}
	l.backup()
	return false
}

func (l *lexer) acceptRun(valid string) {
	for strings.IndexRune(valid, l.next()) >= 0 {
	}
	l.backup()
}

func (l *lexer) errorf(format string, args ...interface{}) stateFn {
	l.items <- item{itemError, l.start, fmt.Sprintf(format, args...)}
	return nil
}

func (l *lexer) nextItem() item {
	item := <-l.items
	l.lastPos = item.pos
	return item
}

func lex(name, input, left, right string) *lexer {
        if left == "" {
                left = leftAngle
        }
        if right == "" {
                right = rightAngle
        }
	l := &lexer{
		name:  name,
		input: input,
                leftAngle: left,
                rightAngle: right,
                leftBracket: left,
                rightBracket: right,
		items: make(chan item),
	}
	go l.run()
	return l
}

func (l *lexer) run() {
	for l.state = lexText; l.state != nil; {
		l.state = l.state(l)
	}
}

const (
	leftAngle = "<"
        rightAngle = ">"
        leftBracket = "["
        rightBracket = "]"
)

func lexText(l *lexer) stateFn {
	for {
		if strings.HasPrefix(l.input[l.pos:], l.leftAngle) {
			if l.pos > l.start {
				l.emit(itemText)
			}
			return lexLeftAngle
		}
		if l.next() == eof {
			break
		}
	}
	// Correctly reached EOF.
	if l.pos > l.start {
		l.emit(itemText)
	}
	l.emit(itemEOF)
	return nil
}

func lexLeftAngle(l *lexer) stateFn {
        l.pos += Pos(len(l.leftAngle))
        l.emit(itemLeftAngleBracket)
        return lexInsideAngle
}

func lexInsideAngle(l *lexer) stateFn {
        switch r := l.next(); {
        case r == eof || isEndOfLine(r):
                return l.errorf("unclosed action")
        case isSpace(r):
                return lexSpace
        case ('0' <= r && r <= '9'):
                l.backup()
                return lexNumber
        default:
                return l.errorf("unrecognized character in action: %#U", r)
        }
        return lexInsideAngle
}

func lexSpace(l *lexer) stateFn {
        for isSpace(l.peek()) {
                l.next()
        }
        l.emit(itemSpace)
        return lexInsideAngle
}

func lexNumber(l *lexer) stateFn {
        l.emit(itemNumber)
        return lexInsideAngle
}

func isSpace(r rune) bool {
        return r == ' ' || r == '\t'
}

func isEndOfLine(r rune) bool {
        return r == '\r' || r == '\n'
}

func isAlphaNumeric(r rune) bool {
        return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}
