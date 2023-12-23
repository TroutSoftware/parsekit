package parsekit

import (
	"io"
	"strings"
	"testing"
)

func TestLexString(t *testing.T) {
	cases := []struct {
		input string
		start int
		match int
	}{
		{`"hello"`, 0, 7},
		{`"handle \" escaped"`, 0, 19},
		// invalid cases
		{`"unterminated`, 0, 13},
		{`'not a string'`, 0, 0},
		// test boundary
		{``, 0, 0},
		{"\"" + strings.Repeat(`lorem ipsum `, 30_000) + "\"", 0, 360_002},
	}

	for _, c := range cases {
		sc := ScanReader(io.NopCloser(strings.NewReader(c.input)))
		sc.offset = c.start // allow arbitrary positioning in the test case
		if got := sc.LexString(); got != c.match {
			t.Errorf("LexString(%s): want %d, got %d", c.input, c.match, got)
		}
	}
}

func TestLexIdents(t *testing.T) {
	cases := []struct {
		input string
		start int
		match int
	}{
		{`myvar`, 0, 5},
		{`my_variable`, 0, 11},
		{`my-variable`, 0, 11},

		{`if else`, 0, 2},
	}

	for _, c := range cases {
		sc := ScanReader(io.NopCloser(strings.NewReader(c.input)))
		sc.offset = c.start // allow arbitrary positioning in the test case
		if got := sc.LexIdent(); got != c.match {
			t.Errorf("LexIdent(%s): want %d, got %d", c.input, c.match, got)
		}
	}
}
