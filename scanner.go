package parsekit

import (
	"encoding"
	"fmt"
	"iter"
	"os"
	"reflect"
	"strconv"
	"unicode/utf8"
)

// Position is a value that represents a source position.
// A position is valid if Line > 0.
type Position struct {
	Filename string // filename, if any
	Offset   int    // byte offset, starting at 0
	Line     int    // line number, starting at 1
	Column   int    // column number, starting at 1 (character count per line)
}

// IsValid reports whether the position is valid.
func (pos *Position) IsValid() bool { return pos.Line > 0 }

func (pos Position) String() string {
	s := pos.Filename
	if s == "" {
		s = "<input>"
	}
	if pos.IsValid() {
		s += fmt.Sprintf(":%d:%d", pos.Line, pos.Column)
	}
	return s
}

// Scanner reads lexemes from a source
type Scanner struct {
	src string

	start, off int

	err error // TODO use this as a way to quickly bail out of parsing
}

// ReadFile reads the content of file name, and passes it to the scanner.
func ReadFile(name string) ParserOptions {
	return func(p *emb) {
		dt, err := os.ReadFile(name)
		if err != nil {
			p.sc = &Scanner{err: err}
			return
		}
		p.sc = &Scanner{src: string(dt)}
	}
}

// ReadString creates a scanner on src.
func ReadString(src string) ParserOptions {
	return func(p *emb) {
		p.sc = &Scanner{src: src}
	}
}

// Tokens returns a stream of Tokens from the underlying scanner.
// The lexer is called repetitively on all yet unread content, and its
// tokens are returned for consumption in the parser.
func (s *Scanner) Tokens(lx Lexer) iter.Seq[Token] {
	return func(yield func(Token) bool) {
		s.start = 0
		for s.off < len(s.src) {
			tk := lx(s)
			if tk != Ignore {
				tk.Lexeme = s.src[s.start:s.off]
				if !yield(tk) {
					return
				}
			}

			s.start = s.off
		}

		yield(EOF)
	}
}

// Advances returns the next character in the stream, and increment the read counter.
func (s *Scanner) Advance() rune {
	if s.off == len(s.src) {
		return utf8.RuneError
	}

	r, sz := utf8.DecodeRuneInString(s.src[s.off:])
	s.off += sz
	return r
}

// Peek returns the next character in the stream, without incrementing the read counter.
func (s *Scanner) Peek() rune {
	if s.off == len(s.src) {
		return utf8.RuneError
	}

	r, _ := utf8.DecodeRuneInString(s.src[s.off:])
	return r
}

// Cursor returns the string currently being scanned
func (s *Scanner) Cursor() string { return string(s.src[s.start:s.off]) }

// EOF is a marker token. The Lexer should return it when [Scanner.Advance] returns an invalid rune.
var EOF Token

// Ignore is a marker token. The Lexer should return it when the current token is to be ignored by the scanner,
// and not passed to the parser.
// This is useful to skip over comments, or empty lines.
var Ignore Token

type Token struct {
	Type  rune
	Value any

	Lexeme string
	Pos    Position
}

func (t Token) Error() error {
	if t.Type != 0 {
		return nil
	}
	return t.Value.(error)
}

// Const returns a constant token
func Const(r rune) Token { return Token{Type: r} }

type Identifier string

// Auto returns a new token with value of type T.
// The value is read from the current lexeme, and converted with:
//
//   - strconv.Unquote for strings if the first character is a quote
//   - the lexeme directly for strings
//   - strconv.ParseInt
//   - unix and iso times for times
//   - calling Unmarshaler otherwise
//
// If the value cannot be parsed, an error token is returned to the parser.
func Auto[T any](r rune, sc *Scanner) Token {

	tt := reflect.TypeFor[T]()
	{
		v := reflect.New(tt).Interface()
		if v, ok := v.(encoding.TextUnmarshaler); ok {
			if err := v.UnmarshalText([]byte(sc.Cursor())); err != nil {
				return Token{Value: err}
			}

			return Token{Type: r, Value: reflect.ValueOf(v).Elem().Interface()}
		}
	}

	switch tt {
	case reflect.TypeFor[string]():
		v, err := strconv.Unquote(sc.Cursor())
		if err != nil {
			return Token{Value: err}
		}
		return Token{Type: r, Value: v}
	case reflect.TypeFor[int]():
		v, err := strconv.ParseInt(sc.Cursor(), 10, 64)
		if err != nil {
			return Token{Value: err}
		}
		return Token{Type: r, Value: v}
	case reflect.TypeFor[error]():
		return Token{Type: r}
	}

	panic("not implemented")
}
