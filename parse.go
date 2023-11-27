// Package parsekit implements a simple, reusable parser for simple grammars.
package parsekit

import (
	"errors"
	"fmt"
)

// Parser implements a recursive descent parser.
// It provides facilities for error reporting, peeking, …
type Parser[T any] struct {
	emb

	peek bool
	tok  rune   // token lookahead
	Lit  string // token literal

	Value  *T
	errors error
}

type emb struct {
	sc      *Scanner
	lx      Lexer
	syncLit []string
}

// ParserOptions specialize the behavior of the parser.
type ParserOptions func(*emb)

// Lexer is a function to create tokens from a scanner.
// lead is the first unicode point of the current token.
// By convention, single-character tokens are represented by their own value (e.g. '{' -> U007B),
// while multiple-character tokens are represented by negative runes (cf the package example).
type Lexer func(sc *Scanner, lead rune) (rune, int)

// ReadFiles is an option to specify which files are to be parsed
func ReadFiles(docs ...string) ParserOptions {
	return func(e *emb) { e.sc = ScanFiles(docs...) }
}

// WithLexer options sets the lexer used by the parser
func WithLexer(lx Lexer) ParserOptions { return func(e *emb) { e.lx = lx } }

// SynchronizeAt sets the synchronisation literals for error recovery.
// See [Parser.Synchronize] for full documentation.
func SynchronizeAt(lits ...string) ParserOptions { return func(c *emb) { c.syncLit = lits } }

// Init creates a new parser.
// At least two options must be provided: (1) a reader, and (2) a lexer function.
// Further options (e.g. [SynchronizeAt])
func Init[T any](opts ...ParserOptions) *Parser[T] {
	var p Parser[T]
	for _, o := range opts {
		o(&p.emb)
	}
	p.Value = new(T)
	return &p
}

// Finish returns the value, and error of the parsing.
// This make it convenient to use at the bottom of a function:
//
//	func ReadConfigFiles() (MyStruct, error) {
//	   p := Init(ReadFiles(xxx), Lexer(yyy))
//	   parseConfig(p)
//	   return p.Finish()
//	}
func (p *Parser[T]) Finish() (*T, error) { return p.Value, p.errors }

// Errf triggers a panic mode with the given formatted error.
// The position is correctly attached to the error.
func (p *Parser[T]) Errf(format string, args ...any) {
	panic(parseError{p.sc.Pos(), fmt.Sprintf(format, args...)})
}

type parseError struct {
	pos Position
	msg string
}

// Error implements error.
func (e parseError) Error() string { return fmt.Sprintf("at %s: %s", e.pos, e.msg) }

const eof = -1
// More returns true if input is left in the stream.
func (p *Parser[T]) More() bool { p.next(); p.peek = true; return p.tok != eof }

func (p *Parser[T]) next() {
	if p.peek {
		p.peek = false
		return
	}

	if p.Lit == ErrLit {
		return
	}

	tk := p.sc.Token()
	var off int
	p.tok, off = p.lx(p.sc, tk)
	p.Lit = string(p.sc.bytes(off))
	p.sc.Advance(off)
}

// ErrLit is the literal value set after a failed call to [Parser.Expect]
const ErrLit = "<error>"

// Expects advances the parser to the next input, making sure it matches the token tk.
func (p *Parser[T]) Expect(tk rune, msg string) {
	p.next()
	if p.tok == tk {
		return
	}
	p.Errf("expected %s, got %q instead", msg, p.Lit)
}

// Match returns true if tk is found at the current parsing point.
// It does not consume any input, so can be used in a test.
func (p *Parser[T]) Match(tk ...rune) bool {
	p.next()
	p.peek = true
	for _, tk := range tk {
		if p.tok == tk {
			p.next()
			return true
		}
	}
	return false
}

// Skip throws away the next token
func (p *Parser[T]) Skip() { p.next() }

// Synchronize handles error recovery in the parsing process:
// when an error occurs, the parser panics all the way to the [Parse.Synchronize] function.
// All tokens are thrown until the first of lits is found
//
// Run this in a top-level `defer` statement in at the level of the synchronisation elements.
func (p *Parser[T]) Synchronize() {
	err := recover()
	if err == nil {
		return
	}
	pe, ok := err.(parseError)
	if !ok {
		panic(pe)
	}

	p.errors = errors.Join(p.errors, pe)
	p.Lit = "" // reset state

	for p.More() {
		p.next()
		for _, slit := range p.syncLit {
			if p.Lit == slit {
				return
			}
		}
	}
}