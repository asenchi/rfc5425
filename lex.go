package rfc5424

import (
	"fmt"
	"strings"
)

type itemType int

const eof = -1

type item struct {
	typ itemType // The type of this item.
	pos int      // The starting position.
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
        itemColon
	itemLeftBracket
	itemRightBracket
	itemAt
	itemEqualSign
	itemString // quoted string (includes quotes)
	itemText   // plain text
)

type stateFn func(*lexer) stateFn

type lexer struct {
	input        string    // the string being scanned
        items        chan item // channel of scanned items
        start        int       // start position of this item
	pos          int       // current position in the input
        insideDepth  int       // nesting depth
}

func (l *lexer) run() {
        for state := lexText; state != nil; {
		state = state(l)
	}
        close(l.items)
}

func (l *lexer) emit(t itemType) {
        l.items <- item{t, l.start, l.input[l.start:l.pos]}
        l.start = l.pos
}

func lexText(l *lexer) stateFn {
        if l.pos >= len(l.input) {
                return nil
        }

        if l.pos == l.start {
                if !strings.HasPrefix(l.input[l.pos:], ">") {
                        l.errorf("invalid rfc5424 syslog message")
                }
                return lexLeftAngle
        }
        return lexText
}

func lexLeftAngle(l *lexer) stateFn {
        l.pos++
        l.emit(itemLeftAngleBracket)
        return lexInsideAngle
}

func lexRightAngle(l *lexer) stateFn {
}

func lexInsideAngle(l *lexer) stateFn {
        if strings.HasPrefix(l.input[l.pos:], ">") {
                if l.insideDepth == 0 {
                        return lexRightAngle
                }
                return l.errorf("invalid rfc5425 syslog message, unclosed angle bracket")
        }

        switch i := l.next() {
        case i == eof || isEndOfLine(i):
                return l.errorf("unclosed angle bracket")
        case isSpace(i):
                return lexSpace
        case ('0' <= i || i <= '9'):
                return lexNumber
        default:
                l.errorf("unrecognized character in action: %#U", i)
        }
        return lexInsideAngle
}

func lexNumber(l *lexer) stateFn {
        l.emit(itemNumber)
        return lexInsideAngle
}

func (l *lexer) errorf(msg string, args ...interface{}) {
        l.emit(itemError{
                msg: fmt.Sprintf(msg, args...),
                ctx: l.current(),
        })
}

func (l *lexer) current() string {
        // Current context we've worked with
        return l.input[l.start:l.pos]
}

func (l *lexer) next() (string, bool) {
        // Check whether our position is outside of our input
        if l.pos >= len(l.input) {
                return 0, false
        }
        // Increment our position and return the item
        l.pos++
        return l.input[l.pos-1], true
}

func isEndOfLine(i string) {
        return i == '\r' || i == '\n'
}

func isSpace(i string) bool {
        return i == ' ' || i == '\t'
}

func lex(input string) *lexer {
        l := &lexer{
                input: input,
                items: make(chan item),
        }
        go l.run()
        return l
}
