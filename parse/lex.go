// go-rst - A reStructuredText parser for Go
// 2014 (c) The go-rst Authors
// MIT Licensed. See LICENSE for details.
package parse

import (
	"fmt"
	"github.com/demizer/go-elog"
	"unicode/utf8"
)

type itemElement int

const (
	itemEOF itemElement = iota
	itemError
	itemTitle
	itemSectionAdornment
	itemParagraph
	itemBlockquote
	itemLiteralBlock
	itemSystemMessage
	itemSpace
	itemBlankLine
)

var elements = [...]string{
	"itemEOF",
	"itemError",
	"itemTitle",
	"itemSectionAdornment",
	"itemParagraph",
	"itemBlockquote",
	"itemLiteralBlock",
	"itemSystemMessage",
	"itemSpace",
	"itemBlankLine",
}

func (t itemElement) String() string { return elements[t] }

var sectionAdornments = []rune{'!', '"', '#', '$', '\'', '%', '&', '(', ')', '*',
	'+', ',', '-', '.', '/', ':', ';', '<', '=', '>', '?', '@', '[', '\\',
	']', '^', '_', '`', '{', '|', '}', '~'}

const EOF rune = -1

type stateFn func(*lexer) stateFn

type item struct {
	ElementName string
	ElementType itemElement
	Position    Pos
	Line        int
	Value       interface{}
}

type systemMessageLevel int

const (
	levelInfo systemMessageLevel = iota
	levelWarning
	levelError
	levelSevere
)

var systemMessageLevels = [...]string{
	"INFO",
	"WARNING",
	"ERROR",
	"SEVERE",
}

func (s systemMessageLevel) String() string { return systemMessageLevels[s] }

type systemMessage struct {
	level  systemMessageLevel
	line   int
	source string
	items  []item
}

type lexer struct {
	name    string
	input   string
	state   stateFn
	pos     Pos
	start   Pos
	width   Pos
	lastPos Pos
	items   chan item
	line    int
}

func lex(name, input string) *lexer {
	l := &lexer{
		name:  name,
		input: input,
		line: 1,
		items: make(chan item),
	}
	go l.run()
	return l
}

// emit passes an item back to the client.
func (l *lexer) emit(t itemElement) {
	log.Debugf("\tEmit %s!\n", t)
	l.items <- item{ElementType: t, ElementName: fmt.Sprint(t),
		Position: l.start, Line: l.line, Value: l.input[l.start:l.pos]}
	l.start = l.pos

}

func (l *lexer) backup() {
	l.pos -= l.width
}

func (l *lexer) current() rune {
	r, _ := utf8.DecodeRuneInString(l.input[l.pos:])
	return r
}

func (l *lexer) previous() rune {
	l.backup()
	r := l.current()
	l.next()
	return r
}

func (l *lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

func (l *lexer) ignore() {
	l.pos += 1
	l.start = l.pos
}

// next returns the next rune in the input.
func (l *lexer) next() rune {
	if int(l.pos) >= len(l.input) {
		log.Debugln("Reached EOF!")
		l.width = 0
		return EOF
	}
	r, w := utf8.DecodeRuneInString(l.input[l.pos:])
	l.width = Pos(w)
	l.pos += l.width
	return r
}

// nextItem returns the next item from the input.
func (l *lexer) nextItem() item {
	item := <-l.items
	l.lastPos = item.Position
	return item

}

func (l *lexer) run() {
	for l.state = lexStart; l.state != nil; {
		l.state = l.state(l)
	}
}

// isSpace reports whether r is a space character.
func isSpace(r rune) bool {
	return r == ' ' || r == '\t'
}

// isEndOfLine reports whether r is an end-of-line character.
func isEndOfLine(r rune) bool {
	return r == '\r' || r == '\n'
}

func lexStart(l *lexer) stateFn {
	log.Debugln("\nTransition lexStart...")
	for {
		if len(l.input) == 0 {
			log.Debugln("\tEmit EOF!")
			l.emit(itemEOF)
			return nil
		}

		log.Debugf("\tlexStart: %q, Start: %d, Pos: %d, Line: %d\n",
			l.input[l.start:l.pos], l.start, l.pos, l.line)

		switch r := l.current(); {
		case isSectionAdornment(r) && isSectionAdornment(l.peek()) && l.pos == 1:
			log.Debugln("Transition lexSection...")
			return lexSection
		case isEndOfLine(r):
			log.Debugln("\tFound newline!")
			l.line += 1
			if isSectionAdornment(l.peek()) {
				log.Debugln("Transition lexSection...")
				return lexSection
			}
			if l.pos > l.start {
				l.emit(itemParagraph)
			}
			l.ignore()
		}
		if l.next() == EOF {
			break
		}
	}

	// Correctly reached EOF.
	if l.pos > l.start {
		l.emit(itemParagraph)
	}

	l.emit(itemEOF)
	return nil
}

func lexSection(l *lexer) stateFn {
	if len(l.input) > 0 {
		log.Debugf("\tlexSection: %q, Pos: %d\n",
			l.input[l.start:l.pos], l.pos)
	}

	if isEndOfLine(l.peek()) {
		l.emit(itemTitle)
		l.ignore()
	}

Loop:
	for {
		switch r := l.next(); {
		case isSectionAdornment(r):
			if len(l.input) > 0 {
				log.Debugf("\tlexSection: %q, Pos: %d\n",
					l.input[l.start:l.pos], l.pos)
			}
		case isEndOfLine(r):
			l.backup()
			l.emit(itemSectionAdornment)
			l.line += 1
			l.ignore()
			break Loop
		}
	}
	return lexStart
}

func isSectionAdornment(r rune) bool {
	for _, a := range sectionAdornments {
		if a == r {
			return true
		}
	}
	return false
}
