// Package parsekit implements a simple, reusable parser for simple grammars.
package parsekit

import (
	"errors"
	"fmt"
	"iter"
)

// Parser implements a recursive descent parser.
// It provides facilities for error reporting, peeking, …
type Parser[T any] struct {
	emb

	next func() (Token, bool)
	stop func()

	peek bool
	tok  Token // token lookahead

	Value  T
	errors error
}

// dedicated type for options in parser – avoid generics in ParserOptions
type emb struct {
	sc *Scanner
	lx Lexer

	syncLit []string
	verbose bool
}

// ParserOptions specialize the behavior of the parser.
type ParserOptions func(*emb)

// Lexer is a stateful function to read tokens from the scanner.
// Each time the function returns, a new token is created, and the scanner advance.
type Lexer func(s *Scanner) Token

// WithLexer options sets the lexer used by the parser
func WithLexer(lx Lexer) ParserOptions { return func(e *emb) { e.lx = lx } }

// SynchronizeAt sets the synchronisation literals for error recovery.
// See [Parser.Synchronize] for full documentation.
func SynchronizeAt(lits ...string) ParserOptions { return func(c *emb) { c.syncLit = lits } }

func Verbose() ParserOptions { return func(e *emb) { e.verbose = true } }

// Init creates a new parser.
// At least two options must be provided: (1) a reader, and (2) a lexer function.
// Further options (e.g. [SynchronizeAt])
func Init[T any](opts ...ParserOptions) *Parser[T] {
	var p Parser[T]
	for _, o := range opts {
		o(&p.emb)
	}

	p.next, p.stop = iter.Pull(p.sc.Tokens(p.lx))

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
func (p *Parser[T]) Finish() (T, error) { return p.Value, p.errors }

// Errf triggers a panic mode with the given formatted error.
// The position is correctly attached to the error.
func (p *Parser[T]) Errf(format string, args ...any) {
	panic(parseError{Position{}, fmt.Sprintf(format, args...)})
}

type parseError struct {
	pos Position
	msg string
}

// Error implements error.
func (e parseError) Error() string { return fmt.Sprintf("at %s: %s", e.pos, e.msg) }

// More returns true if input is left in the stream.
// More does not advance the parser state, so use [Parser.Skip] or [Parser.Expect] to consume a value.
func (p *Parser[T]) More() bool {
	p.lnext()
	p.peek = true
	return p.tok != EOF
}

func prettyrune(r rune) string {
	if r > 0 {
		return fmt.Sprintf("%q", r)
	} else {
		return fmt.Sprintf("%d", r)
	}
}

// ErrLit is the literal value set after a failed call to [Parser.Expect]
const ErrLit = "<error>"

// Expects advances the parser to the next input, making sure it matches the token tk.
func (p *Parser[T]) Expect(tk rune, msg string) {
	p.lnext()
	if p.tok.Type == tk {
		p.peek = false
		return
	}
	p.Errf("expected %s, got %q instead", msg, p.tok)
}

// Match returns true if tk is found at the current parsing point.
// It does not consume any input on failure, so can be used in a test.
func (p *Parser[T]) Match(tk ...rune) bool {
	p.lnext()
	p.peek = true
	for _, tk := range tk {
		if p.tok.Type == tk {
			p.peek = false
			return true
		}
	}
	return false
}

// Skip throws away the current token
func (p *Parser[T]) Skip() {
	if p.peek {
		p.peek = false
		return
	}
	p.lnext()
}

func (p *Parser[T]) lnext() {
	if p.peek {
		return
	}

	p.tok, _ = p.next()
}

func (p *Parser[T]) Lit() string { return p.tok.Lexeme }
func (p *Parser[T]) Val() any    { return p.tok.Value }

// Synchronize handles error recovery in the parsing process:
// when an error occurs, the parser panics all the way to the [Parser.Synchronize] function.
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

	for p.More() {
		for _, slit := range p.syncLit {
			if p.tok.Lexeme == slit {
				return
			}
		}
		p.Skip()
	}
}
