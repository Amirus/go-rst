// go-rst - A reStructuredText parser for Go
// 2014 (c) The go-rst Authors
// MIT Licensed. See LICENSE for details.

package parse

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"
	"unicode/utf8"

	"code.google.com/p/go.text/unicode/norm"

	"github.com/demizer/go-elog"
)

var (
	tEOF = item{Type: itemEOF, StartPosition: 0, Text: ""}
)

func lexTest(t *testing.T, test *Test) []item {
	log.WithFlagsf(log.LdebugFlags, log.Debugf, "Test Path: %s\n", test.path)
	log.WithFlagsf(log.LdebugFlags, log.Debugf,
		"Test Input:\n-----------\n%s\n----------\n", test.data)
	var items []item
	l := lex(test.path, test.data)
	for {
		item := l.nextItem()
		items = append(items, *item)
		if item.Type == itemEOF || item.Type == itemError {
			break
		}
	}
	return items
}

// Test equality between items and expected items from unmarshalled json data,
// field by field.  Returns error in case of error during json unmarshalling,
// or mismatch between items and the expected output.
func equal(t *testing.T, expectItems []item, items []item) {
	var id int
	var found bool
	var pFieldName, eFieldName string
	var pFieldVal, eFieldVal reflect.Value
	var pFieldValS reflect.StructField

	dError := func() {
		var got, exp string
		switch r := pFieldVal.Interface().(type) {
		case ID:
			got = pFieldVal.Interface().(ID).String()
			exp = eFieldVal.Interface().(ID).String()
		case itemElement:
			got = pFieldVal.Interface().(itemElement).String()
			exp = eFieldVal.Interface().(itemElement).String()
		case Line:
			got = pFieldVal.Interface().(Line).String()
			exp = eFieldVal.Interface().(Line).String()
		case StartPosition:
			got = pFieldVal.Interface().(StartPosition).String()
			exp = eFieldVal.Interface().(StartPosition).String()
		case int:
			got = strconv.Itoa(pFieldVal.Interface().(int))
			exp = strconv.Itoa(eFieldVal.Interface().(int))
		case string:
			got = pFieldVal.Interface().(string)
			exp = eFieldVal.Interface().(string)
		default:
			panic(fmt.Errorf("%T is not implemented!", r))
		}
		t.Errorf("\n(ID: %d) Got: %s = %q, Expect: %s = %q\n", id,
			pFieldName, got, eFieldName, exp)
	}

	check := func() {
		if !found {
			t.Errorf("ID: %d does not contain field %q\n", id,
				eFieldName)
			return
		}

		// Handle special cases when comparing fields
		switch eFieldName {
		case "Text":
			normalized := norm.NFC.String(eFieldVal.Interface().(string))
			eFieldVal = reflect.ValueOf(normalized)
		case "StartPosition":
			if pFieldVal.Interface().(StartPosition) == 1 {
				// Ignore StartPositions that begin at 1 from
				// the parsed output items. This allows
				// startPosition to be excluded from the
				// expected items tests (*_items.json).
				return
			}
		}

		if eFieldVal.Interface() != pFieldVal.Interface() {
			dError()
		}
	}

	for eNum, eItem := range expectItems {
		eVal := reflect.ValueOf(eItem)
		pVal := reflect.ValueOf(items[eNum])
		id = int(pVal.FieldByName("ID").Interface().(ID))
		for x := 0; x < eVal.NumField(); x++ {
			eFieldVal = eVal.Field(x)
			eFieldName = eVal.Type().Field(x).Name
			pFieldVal = pVal.FieldByName(eFieldName)
			pFieldValS, found = pVal.Type().FieldByName(eFieldName)
			pFieldName = pFieldValS.Name
			check()
		}
	}

	return
}

var lexerTests = []struct {
	name   string
	input  string
	nIndex int // Expected index after test is run
	nMark  rune
	nWidth int
	nLines int
}{
	{
		name:   "Default 1",
		input:  "Title",
		nIndex: 0, nMark: 'T', nWidth: 1, nLines: 1,
	},
	{
		name:   "Default with diacritic",
		input:  "à Title",
		nIndex: 0, nMark: '\u00E0', nWidth: 2, nLines: 1,
	},
	{
		name:   "Default with two lines",
		input:  "à Title\n=======",
		nIndex: 0, nMark: '\u00E0', nWidth: 2, nLines: 2,
	},
}

func TestLexerNew(t *testing.T) {
	for _, tt := range lexerTests {
		lex := newLexer(tt.name, tt.input)
		if lex.index != tt.nIndex {
			t.Errorf("Test: %q\n\t   "+
				"Got: lexer.index == %d, Expect: %d\n\n",
				lex.name, lex.index, tt.nIndex)
		}
		if lex.mark != tt.nMark {
			t.Errorf("Test: %q\n\t    "+
				"Got: lexer.mark == %#U, Expect: %#U\n\n",
				lex.name, lex.mark, tt.nMark)
		}
		if len(lex.lines) != tt.nLines {
			t.Errorf("Test: %q\n\t    "+
				"Got: lexer.lineNumber == %d, Expect: %d\n\n",
				lex.name, lex.lineNumber(), tt.nLines)
		}
		if lex.width != tt.nWidth {
			t.Errorf("Test: %q\n\t    "+
				"Got: lexer.width == %d, Expect: %d\n\n",
				lex.name, lex.width, tt.nWidth)
		}
	}
}

var lexerGotoLocationTests = []struct {
	name      string
	input     string
	start     int
	startLine int
	lIndex    int // Index of lexer after gotoLocation() is ran
	lMark     rune
	lWidth    int
	lLine     int
}{
	{
		name:  "Goto middle of line",
		input: "Title",
		start: 2, startLine: 1,
		lIndex: 2, lMark: 't', lWidth: 1, lLine: 1,
	},
	{
		name:  "Goto end of line",
		input: "Title",
		start: 5, startLine: 1,
		lIndex: 5, lMark: utf8.RuneError, lWidth: 0, lLine: 1,
	},
}

func TestLexerGotoLocation(t *testing.T) {
	for _, tt := range lexerGotoLocationTests {
		lex := newLexer(tt.name, tt.input)
		lex.gotoLocation(tt.start, tt.startLine)
		if lex.index != tt.lIndex {
			t.Errorf("Test: %q\n\t    "+
				"Got: lex.index == %d, Expect: %d\n\n",
				tt.name, lex.index, tt.lIndex)
		}
		if lex.mark != tt.lMark {
			t.Errorf("Test: %q\n\t    "+
				"Got: lex.mark == %#U, Expect: %#U\n\n",
				tt.name, lex.mark, tt.lMark)
		}
		if lex.width != tt.lWidth {
			t.Errorf("Test: %q\n\t    "+
				"Got: lex.width == %d, Expect: %d\n\n",
				tt.name, lex.width, tt.lWidth)
		}
		if lex.lineNumber() != tt.lLine {
			t.Errorf("Test: %q\n\t    "+
				"Got: lex.line = %d, Expect: %d\n\n",
				tt.name, lex.lineNumber(), tt.lLine)
		}
	}
}

var lexerBackupTests = []struct {
	name      string
	input     string
	start     int
	startLine int
	pos       int // Backup by a number of positions
	lIndex    int // Expected index after backup
	lMark     rune
	lWidth    int
	lLine     int
}{
	{
		name:  "Backup off input",
		input: "Title",
		pos:   1,
		start: 0, startLine: 1,
		lIndex: 0, lMark: 'T', lWidth: 1, lLine: 1, // -1 is EOF
	},
	{
		name:  "Normal Backup",
		input: "Title",
		pos:   2,
		start: 3, startLine: 1,
		lIndex: 1, lMark: 'i', lWidth: 1, lLine: 1,
	},
	{
		name:  "Start after \u00E0",
		input: "à Title",
		pos:   1,
		start: 2, startLine: 1,
		lIndex: 0, lMark: '\u00E0', lWidth: 2, lLine: 1,
	},
	{
		name:  "Backup to previous line",
		input: "Title\n=====",
		pos:   1,
		start: 0, startLine: 2,
		lIndex: 5, lMark: utf8.RuneError, lWidth: 0, lLine: 1,
	},
	{
		name:  "Start after \u00E0, 2nd line",
		input: "Title\nà diacritic",
		pos:   1,
		start: 2, startLine: 2,
		lIndex: 0, lMark: '\u00E0', lWidth: 2, lLine: 2,
	},
	{
		name:  "Backup to previous line newline",
		input: "Title\n\nà diacritic",
		pos:   1,
		start: 0, startLine: 3,
		lIndex: 0, lMark: utf8.RuneError, lWidth: 0, lLine: 2,
	},
	{
		name:  "Backup to end of line",
		input: "Title\n\nà diacritic",
		pos:   1,
		start: 0, startLine: 2,
		lIndex: 5, lMark: utf8.RuneError, lWidth: 0, lLine: 1,
	},
	{
		name:  "Backup 3 byte rune",
		input: "Hello, 世界",
		pos:   1,
		start: 10, startLine: 1,
		lIndex: 7, lMark: '世', lWidth: 3, lLine: 1,
	},
}

func TestLexerBackup(t *testing.T) {
	for _, tt := range lexerBackupTests {
		lex := newLexer(tt.name, tt.input)
		lex.gotoLocation(tt.start, tt.startLine)
		lex.backup(tt.pos)
		if lex.index != tt.lIndex {
			t.Errorf("Test: %q\n\t    "+
				"Got: lex.index == %d, Expect: %d\n\n",
				tt.name, lex.index, tt.lIndex)
		}
		if lex.mark != tt.lMark {
			t.Errorf("Test: %q\n\t    "+
				"Got: lex.mark == %#U, Expect: %#U\n\n",
				tt.name, lex.mark, tt.lMark)
		}
		if lex.width != tt.lWidth {
			t.Errorf("Test: %q\n\t    "+
				"Got: lex.width == %d, Expect: %d\n\n",
				tt.name, lex.width, tt.lWidth)
		}
		if lex.lineNumber() != tt.lLine {
			t.Errorf("Test: %q\n\t    "+
				"Got: lex.line = %d, Expect: %d\n\n",
				tt.name, lex.lineNumber(), tt.lLine)
		}
	}
}

var lexerNextTests = []struct {
	name      string
	input     string
	start     int
	startLine int
	nIndex    int
	nMark     rune
	nWidth    int
	nLine     int
}{
	{
		name:  "next at index 0",
		input: "Title",
		start: 0, startLine: 1,
		nIndex: 1, nMark: 'i', nWidth: 1, nLine: 1,
	},
	{
		name:  "next at index 1",
		input: "Title",
		start: 1, startLine: 1,
		nIndex: 2, nMark: 't', nWidth: 1, nLine: 1,
	},
	{
		name:  "next at end of line",
		input: "Title",
		start: 5, startLine: 1,
		nIndex: 5, nMark: utf8.RuneError, nWidth: 0, nLine: 1,
	},
	{
		name:  "next on diacritic",
		input: "Buy à diacritic",
		start: 4, startLine: 1,
		nIndex: 6, nMark: ' ', nWidth: 1, nLine: 1,
	},
	{
		name:  "next end of 1st line",
		input: "Title\nà diacritic",
		start: 5, startLine: 1,
		nIndex: 0, nMark: '\u00E0', nWidth: 2, nLine: 2,
	},
	{
		name:  "next on 2nd line diacritic",
		input: "Title\nà diacritic",
		start: 0, startLine: 2,
		nIndex: 2, nMark: ' ', nWidth: 1, nLine: 2,
	},
	{
		name:  "next to blank line",
		input: "title\n\nà diacritic",
		start: 5, startLine: 1,
		nIndex: 0, nMark: utf8.RuneError, nWidth: 0, nLine: 2,
	},
	{
		name:  "next on 3 byte rune",
		input: "Hello, 世界",
		start: 7, startLine: 1,
		nIndex: 10, nMark: '界', nWidth: 3, nLine: 1,
	},
	{
		name:  "next on last rune of last line",
		input: "Hello\n\nworld\nyeah!",
		start: 4, startLine: 4,
		nIndex: 5, nMark: utf8.RuneError, nWidth: 0, nLine: 4,
	},
}

func TestLexerNext(t *testing.T) {
	for _, tt := range lexerNextTests {
		lex := newLexer(tt.name, tt.input)
		lex.gotoLocation(tt.start, tt.startLine)
		r, w := lex.next()
		if lex.index != tt.nIndex {
			t.Errorf("Test: %q\n\t    "+
				"Got: lexer.index = %d, Expect: %d\n\n",
				lex.name, lex.index, tt.nIndex)
		}
		if r != tt.nMark {
			t.Errorf("Test: %q\n\t    "+
				"Got: lexer.mark = %#U, Expect: %#U\n\n",
				lex.name, r, tt.nMark)
		}
		if w != tt.nWidth {
			t.Errorf("Test: %q\n\t    "+
				"Got: lexer.width = %d, Expect: %d\n\n",
				lex.name, w, tt.nWidth)
		}
		if lex.lineNumber() != tt.nLine {
			t.Errorf("Test: %q\n\t    "+
				"Got: lexer.line = %d, Expect: %d\n\n",
				lex.name, lex.lineNumber(), tt.nLine)
		}
	}
}

var lexerPeekTests = []struct {
	name      string
	input     string
	start     int // Start position begins at 0
	startLine int // Begins at 1
	lIndex    int // l* fields do not change after peek() is called
	lMark     rune
	lWidth    int
	lLine     int
	pMark     rune // p* are the expected return values from peek()
	pWidth    int
}{
	{
		name:  "Peek start at 0",
		input: "Title",
		start: 0, startLine: 1,
		lIndex: 0, lMark: 'T', lWidth: 1, lLine: 1,
		pMark: 'i', pWidth: 1,
	},
	{
		name:  "Peek start at 1",
		input: "Title",
		start: 1, startLine: 1,
		lIndex: 1, lMark: 'i', lWidth: 1, lLine: 1,
		pMark: 't', pWidth: 1,
	},
	{
		name:  "Peek start at diacritic",
		input: "à Title",
		start: 0, startLine: 1,
		lIndex: 0, lMark: '\u00E0', lWidth: 2, lLine: 1,
		pMark: ' ', pWidth: 1,
	},
	{
		name:  "Peek starting on 2nd line",
		input: "Title\nà diacritic",
		start: 0, startLine: 2,
		lIndex: 0, lMark: '\u00E0', lWidth: 2, lLine: 2,
		pMark: ' ', pWidth: 1,
	},
	{
		name:  "Peek starting on blank line",
		input: "Title\n\nà diacritic",
		start: 0, startLine: 2,
		lIndex: 0, lMark: utf8.RuneError, lWidth: 0, lLine: 2,
		pMark: '\u00E0', pWidth: 2,
	},
	{
		name:  "Peek with 3 byte rune",
		input: "Hello, 世界",
		start: 7, startLine: 1,
		lIndex: 7, lMark: '世', lWidth: 3, lLine: 1,
		pMark: '界', pWidth: 3,
	},
}

func TestLexerPeek(t *testing.T) {
	for _, tt := range lexerPeekTests {
		lex := newLexer(tt.name, tt.input)
		lex.gotoLocation(tt.start, tt.startLine)
		r := lex.peek()
		w := utf8.RuneLen(r)
		if lex.index != tt.lIndex {
			t.Errorf("Test: %q\n\t    "+
				"Got: lexer.index == %d, Expect: %d\n\n",
				lex.name, lex.index, tt.lIndex)
		}
		if lex.width != tt.lWidth {
			t.Errorf("Test: %q\n\t    "+
				"Got: lexer.width == %d, Expect: %d\n\n",
				lex.name, lex.width, tt.lWidth)
		}
		if lex.lineNumber() != tt.lLine {
			t.Errorf("Test: %q\n\t    "+
				"Got: lexer.line = %d, Expect: %d\n\n",
				lex.name, lex.lineNumber(), tt.lLine)
		}
		if r != tt.pMark {
			t.Errorf("Test: %q\n\t    "+
				"Got: peek().rune  == %q, Expect: %q\n\n",
				lex.name, r, tt.pMark)
		}
		if w != tt.pWidth {
			t.Errorf("Test: %q\n\t    "+
				"Got: peek().width == %d, Expect: %d\n\n",
				lex.name, w, tt.pWidth)
		}
	}
}

func TestLexerIsLastLine(t *testing.T) {
	input := "==============\nTitle\n=============="
	lex := newLexer("isLastLine test 1", input)
	lex.gotoLocation(0, 1)
	if lex.isLastLine() != false {
		t.Errorf("Test: %q\n\t    "+
			"Got: isLastLine == %t, Expect: %t\n\n",
			lex.name, lex.isLastLine(), false)
	}
	lex = newLexer("isLastLine test 2", input)
	lex.gotoLocation(0, 2)
	if lex.isLastLine() != false {
		t.Errorf("Test: %q\n\t    "+
			"Got: isLastLine == %t, Expect: %t\n\n",
			lex.name, lex.isLastLine(), false)
	}
	lex = newLexer("isLastLine test 3", input)
	lex.gotoLocation(0, 3)
	if lex.isLastLine() != true {
		t.Errorf("Test: %q\n\t    "+
			"Got: isLastLine == %t, Expect: %t\n\n",
			lex.name, lex.isLastLine(), true)
	}
}

var peekNextLineTests = []struct {
	name      string
	input     string
	start     int
	startLine int
	lIndex    int // l* fields do not change after peekNextLine() is called
	lLine     int
	nText     string
}{
	{
		name:  "Get next line after first",
		input: "==============\nTitle\n==============",
		start: 0, startLine: 1,
		lIndex: 0, lLine: 1, nText: "Title",
	},
	{
		name:  "Get next line after second.",
		input: "==============\nTitle\n==============",
		start: 0, startLine: 2,
		lIndex: 0, lLine: 2, nText: "==============",
	},
	{
		name:  "Get next line from middle of first",
		input: "==============\nTitle\n==============",
		start: 5, startLine: 1,
		lIndex: 5, lLine: 1, nText: "Title",
	},
	{
		name:  "Attempt to get next line after last",
		input: "==============\nTitle\n==============",
		start: 5, startLine: 3,
		lIndex: 5, lLine: 3, nText: "",
	},
	{
		name:  "Peek to a blank line",
		input: "==============\n\nTitle\n==============",
		start: 5, startLine: 1,
		lIndex: 5, lLine: 1, nText: "",
	},
}

func TestLexerPeekNextLine(t *testing.T) {
	for _, tt := range peekNextLineTests {
		lex := newLexer(tt.name, tt.input)
		lex.gotoLocation(tt.start, tt.startLine)
		out := lex.peekNextLine()
		if lex.index != tt.lIndex {
			t.Errorf("Test: %q\n\t    "+
				"Got: lexer.index == %d, Expect: %d\n\n",
				lex.name, lex.index, tt.lIndex)
		}
		if lex.lineNumber() != tt.lLine {
			t.Errorf("Test: %q\n\t    "+
				"Got: lexer.line = %d, Expect: %d\n\n",
				lex.name, lex.lineNumber(), tt.lLine)
		}
		if out != tt.nText {
			t.Errorf("Test: %q\n\t    "+
				"Got: text == %s, Expect: %s\n\n",
				lex.name, out, tt.nText)
		}
	}
}

func TestLexId(t *testing.T) {
	testPath := testPathFromName("00.00-title-paragraph")
	test := LoadLexTest(t, testPath)
	items := lexTest(t, test)
	if items[0].IDNumber() != 1 {
		t.Error("ID != 1")
	}
	if items[0].ID.String() != "1" {
		t.Error(`String ID != "1"`)
	}
}

func TestLexLine(t *testing.T) {
	testPath := testPathFromName("00.00-title-paragraph")
	test := LoadLexTest(t, testPath)
	items := lexTest(t, test)
	if items[0].LineNumber() != 1 {
		t.Error("Line != 1")
	}
	if items[0].Line.String() != "1" {
		t.Error(`String Line != "1"`)
	}
}

func TestLexStartPosition(t *testing.T) {
	testPath := testPathFromName("00.00-title-paragraph")
	test := LoadLexTest(t, testPath)
	items := lexTest(t, test)
	if items[0].Position() != 1 {
		t.Error("StartPosition != 1")
	}
	if items[0].StartPosition.String() != "1" {
		t.Error(`String StartPosition != "1"`)
	}
}
