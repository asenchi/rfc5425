package rfc5424

import (
	"fmt"
	"testing"
)

var itemName = map[itemType]string{
	itemError:             "error",
	itemEOF:               "EOF",
	itemLeftAngleBracket:  "<",
	itemRightAngleBracket: ">",
	itemNumber:            "number",
	itemSpace:             "space",
	itemHyphen:            "-",
	itemLeftBracket:       "[",
	itemRightBracket:      "]",
	itemNil:               "nil",
}

func (i itemType) String() string {
	s := itemName[i]
	if s == "" {
		return fmt.Sprintf("item%d", int(i))
	}
	return s
}

type lexTest struct {
	name  string
	input string
	items []item
}

var (
	tEOF   = item{itemEOF, 0, ""}
	tLang  = item{itemLeftAngleBracket, 0, "<"}
	tRang  = item{itemRightAngleBracket, 0, ">"}
	tLbra  = item{itemLeftBracket, 0, "["}
	tRbra  = item{itemLeftBracket, 0, "]"}
	tSpace = item{itemSpace, 0, " "}
	tHyph  = item{itemHyphen, 0, "-"}
)

var lexTests = []lexTest{
	{"empty", "", []item{tEOF}},
	{"anglebrackets", "<86>", []item{
		tLang,
		{itemNumber, 0, "8"},
		{itemNumber, 0, "6"},
		tRang,
		tEOF,
	}},
}

func collect(t *lexTest, left, right string) (items []item) {
	l := lex(t.name, t.input, left, right)
	for {
		item := l.nextItem()
		items = append(items, item)
		if item.typ == itemEOF || item.typ == itemError {
			break
		}
	}
	return
}

func equal(i1, i2 []item, checkPos bool) bool {
	if len(i1) != len(i2) {
		return false
	}
	for k := range i1 {
		if i1[k].typ != i2[k].typ {
			return false
		}
		if i1[k].val != i2[k].val {
			return false
		}
		if checkPos && i1[k].pos != i2[k].pos {
			return false
		}
	}
	return true
}

func TestLex(t *testing.T) {
	for _, test := range lexTests {
		items := collect(&test, "", "")
		if !equal(items, test.items, false) {
			t.Errorf("%s: got\n\t%+v\nexpected\n\t%v", test.name, items, test.items)
		}
	}
}
