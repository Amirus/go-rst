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
	log.Debugf("Test Path: %s\n", test.path)
	log.Debugf("Test Input:\n-----------\n%s\n----------\n", test.data)
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

// Test equality between items and expected items from unmarshalled json data, field by field.
// Returns error in case of error during json unmarshalling, or mismatch between items and the
// expected output.
func equal(t *testing.T, items []item, expectItems []item) {
	var id int
	var found bool
	var pFieldName, eFieldName string
	var pFieldVal, eFieldVal reflect.Value
	var pFieldValStruct reflect.StructField

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
		t.Errorf("\n(ID: %d) Got: %s = %q, Expect: %s = %q\n", id, pFieldName, got,
			eFieldName, exp)
	}

	for eNum, eItem := range expectItems {
		eVal := reflect.ValueOf(eItem)
		pVal := reflect.ValueOf(items[eNum])
		id = int(pVal.FieldByName("ID").Interface().(ID))
		for x := 0; x < eVal.NumField(); x++ {
			eFieldVal = eVal.Field(x)
			eFieldName = eVal.Type().Field(x).Name
			pFieldVal = pVal.FieldByName(eFieldName)
			pFieldValStruct, found = pVal.Type().FieldByName(eFieldName)
			pFieldName = pFieldValStruct.Name
			if !found {
				t.Errorf("Parsed item (ID: %d) does not contain field %q\n", id,
					eFieldName)
				continue
			} else if eFieldName == "Text" {
				if eFieldVal.Interface() == nil {
					continue
				}
				if pFieldVal.Interface() !=
					norm.NFC.String(eFieldVal.Interface().(string)) {
					dError()
				}
			} else if pFieldVal.Interface() != eFieldVal.Interface() {
				dError()
			}
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
			t.Errorf("Test: %s\n\t Got: lexer.index == %d, Expect: %d\n\n",
				lex.name, lex.index, tt.nIndex)
		}
		if lex.mark != tt.nMark {
			t.Errorf("Test: %s\n\t Got: lexer.mark == %#U, Expect: %#U\n\n",
				lex.name, lex.mark, tt.nMark)
		}
		if len(lex.lines) != tt.nLines {
			t.Errorf("Test: %s\n\t Got: lexer.lineNumber == %d, Expect: %d\n\n",
				lex.name, lex.lineNumber(), tt.nLines)
		}
		if lex.width != tt.nWidth {
			t.Errorf("Test: %s\n\t Got: lexer.width == %d, Expect: %d\n\n",
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
			t.Errorf("Test: %s\n\t Got: lex.index == %d, Expect: %d\n\n",
				tt.name, lex.index, tt.lIndex)
		}
		if lex.mark != tt.lMark {
			t.Errorf("Test: %s\n\t Got: lex.mark == %#U, Expect: %#U\n\n",
				tt.name, lex.mark, tt.lMark)
		}
		if lex.width != tt.lWidth {
			t.Errorf("Test: %s\n\t Got: lex.width == %d, Expect: %d\n\n",
				tt.name, lex.width, tt.lWidth)
		}
		if lex.lineNumber() != tt.lLine {
			t.Errorf("Test: %s\n\t Got: lex.line = %d, Expect: %d\n\n",
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
			t.Errorf("Test: %s\n\t Got: lex.index == %d, Expect: %d\n\n",
				tt.name, lex.index, tt.lIndex)
		}
		if lex.mark != tt.lMark {
			t.Errorf("Test: %s\n\t Got: lex.mark == %#U, Expect: %#U\n\n",
				tt.name, lex.mark, tt.lMark)
		}
		if lex.width != tt.lWidth {
			t.Errorf("Test: %s\n\t Got: lex.width == %d, Expect: %d\n\n",
				tt.name, lex.width, tt.lWidth)
		}
		if lex.lineNumber() != tt.lLine {
			t.Errorf("Test: %s\n\t Got: lex.line = %d, Expect: %d\n\n",
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
			t.Errorf("Test: %s\n\t Got: lexer.index = %d, Expect: %d\n\n",
				lex.name, lex.index, tt.nIndex)
		}
		if r != tt.nMark {
			t.Errorf("Test: %s\n\t Got: lexer.mark = %#U, Expect: %#U\n\n",
				lex.name, r, tt.nMark)
		}
		if w != tt.nWidth {
			t.Errorf("Test: %s\n\t Got: lexer.width = %d, Expect: %d\n\n",
				lex.name, w, tt.nWidth)
		}
		if lex.lineNumber() != tt.nLine {
			t.Errorf("Test: %s\n\t Got: lexer.line = %d, Expect: %d\n\n",
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
		r, w := lex.peek()
		if lex.index != tt.lIndex {
			t.Errorf("Test: %s\n\t Got: lexer.index == %d, Expect: %d\n\n",
				lex.name, lex.index, tt.lIndex)
		}
		if lex.width != tt.lWidth {
			t.Errorf("Test: %s\n\t Got: lexer.width == %d, Expect: %d\n\n",
				lex.name, lex.width, tt.lWidth)
		}
		if lex.lineNumber() != tt.lLine {
			t.Errorf("Test: %s\n\t Got: lexer.line = %d, Expect: %d\n\n",
				lex.name, lex.lineNumber(), tt.lLine)
		}
		if r != tt.pMark {
			t.Errorf("Test: %s\n\t Got: peek().rune  == %q, Expect: %q\n\n",
				lex.name, r, tt.pMark)
		}
		if w != tt.pWidth {
			t.Errorf("Test: %s\n\t Got: peek().width == %d, Expect: %d\n\n",
				lex.name, w, tt.pWidth)
		}
	}
}

func TestLexerIsLastLine(t *testing.T) {
	input := "==============\nTitle\n=============="
	lex := newLexer("isLastLine test 1", input)
	lex.gotoLocation(0, 1)
	if lex.isLastLine() != false {
		t.Errorf("Test: %s\n\t Got: isLastLine == %t, Expect: %t\n\n",
			lex.name, lex.isLastLine(), false)
	}
	lex = newLexer("isLastLine test 2", input)
	lex.gotoLocation(0, 2)
	if lex.isLastLine() != false {
		t.Errorf("Test: %s\n\t Got: isLastLine == %t, Expect: %t\n\n",
			lex.name, lex.isLastLine(), false)
	}
	lex = newLexer("isLastLine test 3", input)
	lex.gotoLocation(0, 3)
	if lex.isLastLine() != true {
		t.Errorf("Test: %s\n\t Got: isLastLine == %t, Expect: %t\n\n",
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
			t.Errorf("Test: %s\n\t Got: lexer.index == %d, Expect: %d\n\n",
				lex.name, lex.index, tt.lIndex)
		}
		if lex.lineNumber() != tt.lLine {
			t.Errorf("Test: %s\n\t Got: lexer.line = %d, Expect: %d\n\n",
				lex.name, lex.lineNumber(), tt.lLine)
		}
		if out != tt.nText {
			t.Errorf("Test: %s\n\t Got: text == %s, Expect: %s\n\n",
				lex.name, out, tt.nText)
		}
	}
}

func TestLexId(t *testing.T) {
	testPath := "test_section/01_title_good/00.00_title_paragraph"
	test := LoadTest(testPath)
	items := lexTest(t, test)
	if items[0].IDNumber() != 1 {
		t.Error("ID != 1")
	}
	if items[0].ID.String() != "1" {
		t.Error(`String ID != "1"`)
	}
}

func TestLexLine(t *testing.T) {
	testPath := "test_section/01_title_good/00.00_title_paragraph"
	test := LoadTest(testPath)
	items := lexTest(t, test)
	if items[0].LineNumber() != 1 {
		t.Error("Line != 1")
	}
	if items[0].Line.String() != "1" {
		t.Error(`String Line != "1"`)
	}
}

func TestLexStartPosition(t *testing.T) {
	testPath := "test_section/01_title_good/00.00_title_paragraph"
	test := LoadTest(testPath)
	items := lexTest(t, test)
	if items[0].Position() != 1 {
		t.Error("StartPosition != 1")
	}
	if items[0].StartPosition.String() != "1" {
		t.Error(`String StartPosition != "1"`)
	}
}

func TestLexSectionTitleGood0000(t *testing.T) {
	// Basic title, underline, blankline, and paragraph test
	testPath := "test_section/01_title_good/00.00_title_paragraph"
	test := LoadTest(testPath)
	items := lexTest(t, test)
	equal(t, items, test.expectItems())
}

func TestLexSectionTitleGood0001(t *testing.T) {
	// Basic title, underline, and paragraph with no blankline line after the
	// section.
	testPath := "test_section/01_title_good/00.01_paragraph_noblankline"
	test := LoadTest(testPath)
	items := lexTest(t, test)
	equal(t, items, test.expectItems())
}

func TestLexSectionTitleGood0002(t *testing.T) {
	// A title that begins with a combining unicode character \u0301. Tests to
	// make sure the 2 byte unicode does not contribute to the underline length
	// calculation.
	testPath := "test_section/01_title_good/00.02_title_combining_chars"
	test := LoadTest(testPath)
	items := lexTest(t, test)
	equal(t, items, test.expectItems())
}

func TestLexSectionTitleGood0100(t *testing.T) {
	// A basic section in between paragraphs.
	testPath := "test_section/01_title_good/01.00_para_head_para"
	test := LoadTest(testPath)
	items := lexTest(t, test)
	equal(t, items, test.expectItems())
}

func TestLexSectionTitleGood0200(t *testing.T) {
	// Tests section parsing on 3 character long title and underline.
	testPath := "test_section/01_title_good/02.00_short_title"
	test := LoadTest(testPath)
	items := lexTest(t, test)
	equal(t, items, test.expectItems())
}

func TestLexSectionTitleGood0300(t *testing.T) {
	// Tests a single section with no other element surrounding it.
	testPath := "test_section/01_title_good/03.00_empty_section"
	test := LoadTest(testPath)
	items := lexTest(t, test)
	equal(t, items, test.expectItems())
}

func TestLexSectionTitleBad0000(t *testing.T) {
	// Tests for severe system messages when the sections are indented.
	testPath := "test_section/02_title_bad/00.00_unexpected_titles"
	test := LoadTest(testPath)
	items := lexTest(t, test)
	equal(t, items, test.expectItems())
}

func TestLexSectionTitleBad0100(t *testing.T) {
	// Tests for severe system message on short title underline
	testPath := "test_section/02_title_bad/01.00_short_underline"
	test := LoadTest(testPath)
	items := lexTest(t, test)
	equal(t, items, test.expectItems())
}

func TestLexSectionTitleBad0200(t *testing.T) {
	// Tests for title underlines that are less than three characters.
	testPath := "test_section/02_title_bad/02.00_short_title_short_underline"
	test := LoadTest(testPath)
	items := lexTest(t, test)
	equal(t, items, test.expectItems())
}

func TestLexSectionTitleBad0201(t *testing.T) {
	// Tests for title overlines and underlines that are less than three characters.
	testPath := "test_section/02_title_bad/02.01_short_title_short_overline_and_underline"
	test := LoadTest(testPath)
	items := lexTest(t, test)
	equal(t, items, test.expectItems())
}

func TestLexSectionTitleBad0202(t *testing.T) {
	// Tests for short title overline with missing underline when the overline
	// is less than three characters.
	testPath := "test_section/02_title_bad/02.02_short_title_short_overline_missing_underline"
	test := LoadTest(testPath)
	items := lexTest(t, test)
	equal(t, items, test.expectItems())
}

func TestLexSectionLevelGood0000(t *testing.T) {
	// Tests section level return to level one after three subsections.
	testPath := "test_section/03_level_good/00.00_section_level_return"
	test := LoadTest(testPath)
	items := lexTest(t, test)
	equal(t, items, test.expectItems())
}

func TestLexSectionLevelGood0001(t *testing.T) {
	// Tests section level return to level one after 1 subsection. The second
	// level one section has one subsection.
	testPath := "test_section/03_level_good/00.01_section_level_return"
	test := LoadTest(testPath)
	items := lexTest(t, test)
	equal(t, items, test.expectItems())
}

func TestLexSectionLevelGood0002(t *testing.T) {
	// Test section level with subsection 4 returning to level two.
	testPath := "test_section/03_level_good/00.02_section_level_return"
	test := LoadTest(testPath)
	items := lexTest(t, test)
	equal(t, items, test.expectItems())
}

func TestLexSectionLevelGood0100(t *testing.T) {
	// Tests section level return with title overlines
	testPath := "test_section/03_level_good/01.00_section_level_return"
	test := LoadTest(testPath)
	items := lexTest(t, test)
	equal(t, items, test.expectItems())
}

func TestLexSectionLevelBad0000(t *testing.T) {
	// Test section level return on bad level 2 section adornment
	testPath := "test_section/04_level_bad/00.00_bad_subsection_order"
	test := LoadTest(testPath)
	items := lexTest(t, test)
	equal(t, items, test.expectItems())
}

func TestLexSectionLevelBad0001(t *testing.T) {
	// Test section level return with title overlines on bad level 2 section adornment
	testPath := "test_section/04_level_bad/00.01_bad_subsection_order_with_overlines"
	test := LoadTest(testPath)
	items := lexTest(t, test)
	equal(t, items, test.expectItems())
}

func TestLexSectionTitleWithOverlineGood0000(t *testing.T) {
	// Test simple section with title overline.
	testPath := "test_section/05_title_with_overline_good/00.00_title_overline"
	test := LoadTest(testPath)
	items := lexTest(t, test)
	equal(t, items, test.expectItems())
}

func TestLexSectionTitleWithOverlineGood0100(t *testing.T) {
	// Test simple section with inset title and overline.
	testPath := "test_section/05_title_with_overline_good/01.00_inset_title_with_overline"
	test := LoadTest(testPath)
	items := lexTest(t, test)
	equal(t, items, test.expectItems())
}

func TestLexSectionTitleWithOverlineGood0200(t *testing.T) {
	// Test sections with three character adornments lines.
	testPath := "test_section/05_title_with_overline_good/02.00_three_char_section_title"
	test := LoadTest(testPath)
	items := lexTest(t, test)
	equal(t, items, test.expectItems())
}

func TestLexSectionTitleWithOverlineBad0000(t *testing.T) {
	// Test section title with overline, but no underline.
	testPath := "test_section/06_title_with_overline_bad/00.00_inset_title_missing_underline"
	test := LoadTest(testPath)
	items := lexTest(t, test)
	equal(t, items, test.expectItems())
}

func TestLexSectionTitleWithOverlineBad0001(t *testing.T) {
	// Test inset title with overline but missing underline.
	testPath := "test_section/06_title_with_overline_bad/00.01_inset_title_missing_underline_with_blankline"
	test := LoadTest(testPath)
	items := lexTest(t, test)
	equal(t, items, test.expectItems())
}

func TestLexSectionTitleWithOverlineBad0002(t *testing.T) {
	// Test inset title with overline but missing underline. The title is
	// followed by a blank line and a paragraph.
	testPath := "test_section/06_title_with_overline_bad/00.02_inset_title_missing_underline_and_para"
	test := LoadTest(testPath)
	items := lexTest(t, test)
	equal(t, items, test.expectItems())
}

func TestLexSectionTitleWithOverlineBad0003(t *testing.T) {
	// Test section overline with missmatched underline.
	testPath := "test_section/06_title_with_overline_bad/00.03_inset_title_mismatched_underline"
	test := LoadTest(testPath)
	items := lexTest(t, test)
	equal(t, items, test.expectItems())
}

func TestLexSectionTitleWithOverlineBad0100(t *testing.T) {
	// Test overline with really long title.
	testPath := "test_section/06_title_with_overline_bad/01.00_title_too_long"
	test := LoadTest(testPath)
	items := lexTest(t, test)
	equal(t, items, test.expectItems())
}

func TestLexSectionTitleWithOverlineBad0200(t *testing.T) {
	// Test overline and underline with blanklines instead of a title.
	testPath := "test_section/06_title_with_overline_bad/02.00_missing_titles_with_blankline"
	test := LoadTest(testPath)
	items := lexTest(t, test)
	equal(t, items, test.expectItems())
}

func TestLexSectionTitleWithOverlineBad0201(t *testing.T) {
	// Test overline and underline with nothing where the title is supposed to
	// be.
	testPath := "test_section/06_title_with_overline_bad/02.01_missing_titles_with_noblankline"
	test := LoadTest(testPath)
	items := lexTest(t, test)
	equal(t, items, test.expectItems())
}

// func TestLexSectionTitleWithOverlineBad0300(t *testing.T) {
// testPath := "test_section/06_title_with_overline_bad/03.00_incomplete_section"
// test := LoadTest(testPath)
// items := lexTest(t, test)
// equal(t, items, test.expectItems())
// }

// func TestLexSectionTitleWithOverlineBad0301(t *testing.T) {
// testPath := // "test_section/06_title_with_overline_bad/03.01_incomplete_sections_no_title"
// test := LoadTest(testPath)
// items := lexTest(t, test)
// equal(t, items, test.expectItems())
// }

// func TestLexSectionTitleWithOverlineBad0400(t *testing.T) {
// testPath := // "test_section/06_title_with_overline_bad/04.00_indented_title_short_overline_and_underline"
// test := LoadTest(testPath)
// items := lexTest(t, test)
// equal(t, items, test.expectItems())
// }

// func TestLexSectionTitleWithOverlineBad0500(t *testing.T) {
// testPath := // "test_section/06_title_with_overline_bad/05.00_two_char_section_title"
// test := LoadTest(testPath)
// items := lexTest(t, test)
// equal(t, items, test.expectItems())
// }

// func TestLexSectionTitleNumberedGood0000(t *testing.T) {
// testPath := // // "test_section/07_title_numbered_good/00.00_numbered_title"
// test := LoadTest(testPath)
// items := lexTest(t, test)
// equal(t, items, test.expectItems())
// }

// func TestLexSectionTitleNumberedGood0100(t *testing.T) {
// testPath := // // "test_section/07_title_numbered_good/01.00_enum_list_with_numbered_title"
// test := LoadTest(testPath)
// items := lexTest(t, test)
// equal(t, items, test.expectItems())
// }

// func TestLexSectionTitleWithInlineMarkupGood0000(t *testing.T) {
// testPath := // "test_section/08_title_with_inline_markup_good/00.00_title_with_inline_markup"
// test := LoadTest(testPath)
// items := lexTest(t, test)
// equal(t, items, test.expectItems())
// }
