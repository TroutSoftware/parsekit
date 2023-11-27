package parsekit

import (
	"fmt"
	"io"
	"os"
	"unicode/utf8"
)

// objectives:
//  - fast scanner based on iopipe (0 alloc)
//  - position handling
//  - state machine for text, identifiers and integers
//  - UTF-8 compliant

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

const (
	bufLen      = 8192 // at least utf8.UTFMax
	minReadSize = bufLen >> 2
)

// zero-alloc byte reader from underlying stream.
// Original idea: https://dave.cheney.net/high-performance-json.html#_reading
//
// TODO: evaluate if we need to shrink the reader
// TODO: align windows at utf8 boundaries
type byteReader struct {
	data []byte
	off  int
	r    io.ReadCloser
	err  error
}

func (b *byteReader) release(n int)    { b.off += n }
func (b *byteReader) window() []byte   { return b.data[b.off:] }
func (b *byteReader) rearview() []byte { return b.data[:b.off] }
func (b *byteReader) extend() int {
	if b.err != nil || b.r == nil {
		return 0
	}

	remaining := len(b.data) - b.off
	if remaining == 0 {
		b.data = b.data[:0]
		b.off = 0
	}

	if cap(b.data)-len(b.data) >= minReadSize {
		// enough space
	} else if cap(b.data)-remaining >= minReadSize {
		b.compact() // move data to front
	} else {
		b.grow() // allocate more
	}
	remaining += b.off
	n, err := b.r.Read(b.data[remaining:cap(b.data)])
	b.data = b.data[:remaining+n]
	b.err = err
	return n
}

func (b *byteReader) grow() {
	buf := make([]byte, max(cap(b.data)*2, bufLen))
	copy(buf, b.data[b.off:])
	b.data = buf
	b.off = 0
}

func (b *byteReader) compact() {
	copy(b.data, b.data[b.off:])
	b.off = 0
}

// Scanner reads tokens from a stream of multiple files.
// It efficiently tracks position information.
type Scanner struct {
	br byteReader

	files, last  *file
	line, offset int
}

type file struct {
	Name string
	next *file
}

// Error exposes the underlying [io.Reader] error
func (sc *Scanner) Error() error { return sc.br.err }

// ScanFiles create a scanner over files with names.
// Files are scanned in the order they are given in, and no token can span two files.
func ScanFiles(names ...string) *Scanner {
	var sc Scanner
	var last *file
	for _, f := range names {
		if last == nil {
			sc.files = &file{Name: f}
			last = sc.files
		} else {
			last.next = &file{Name: f}
			last = last.next
		}
	}
	sc.last = &file{next: sc.files} // placeholder to initalize the read
	sc.br.err = io.EOF
	return &sc
}

// Token reads the next character in the stream, skipping white spaces.
func (s *Scanner) Token() rune {
	w := s.br.window()
	blk := 0

fLoop:
	for {
		for i, c := range w {
			switch c {
			case '\n':
				s.line++
				fallthrough
			case ' ', '\r', '\t':
				blk++
				continue
			}

			s.br.release(blk)
			r, _ := utf8.DecodeRune(w[i:])
			return r
		}

		if s.br.extend() == 0 {
			// eof
			if s.last.next == nil {
				return 0
			}
			if s.br.r != nil {
				s.br.r.Close() // no error check, only reading
			}

			s.last = s.last.next
			s.br.r, s.br.err = os.Open(s.last.Name)
			s.br.extend()
			w = s.br.window()
			s.offset, s.line = 0, 1
			continue fLoop
		}
	}
}

// CatchAll is a well-known transition that get applied if no other transition matches.
const CatchAll = 0

// ScanWithTable uses the state transitions table to read the next characters.
// A transition table consists, for each state, of the pair (token, next_state).
// Transitions table can be constructed from a readable textual definition using the transgen.awk script.
func (s *Scanner) ScanWithTable(transitions [][]byte) (prev uint8, n int) {
	offset := 0
	state, end := uint8(0), uint8(len(transitions)) // by construction of the caller
	w := s.br.window()
winLoop:
	for _, elem := range w[offset:] {
		tt := transitions[state]
		for i := 0; i < len(tt); i += 2 {
			if tt[i] == elem {
				nstate := tt[i+1]
				if nstate == end {
					return state, offset
				}
				offset++
				state = nstate
				continue winLoop
			}
		}
		if tt[0] == CatchAll {
			nstate := tt[1]
			if nstate == end {
				return state, offset
			}
			offset++
			state = nstate
			continue winLoop
		}
		return state, offset
	}

	// need more data
	if s.br.extend() == 0 {
		return state, offset
	}
	w = s.br.window()
	goto winLoop
}

func (s *Scanner) bytes(n int) []byte { return s.br.window()[:n] }

// Advance moves the cursor by n bytes.
func (s *Scanner) Advance(n int) { s.br.release(n); s.offset += n }

// Pos returns the parser position, including the current column.
func (s *Scanner) Pos() Position {
	col := 0
	w := s.br.rearview()
	for i := len(w); i > 0; {
		r, sz := utf8.DecodeLastRune(w[:i])
		col++
		if r == '\n' {
			break
		}
		i -= sz
	}

	return Position{
		Filename: s.last.Name,
		Line:     s.line,
		Offset:   s.offset,
		Column:   col, // note this might not work with too long lines, probably OK
	}
}
