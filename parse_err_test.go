package parsekit

import (
	"testing"
	"unicode/utf8"
)

func TestErrMessage(t *testing.T) {
	const txt = `option "color" "red"
	opton "destination" "Turin"
	option "time" "3h"
	`

	p := Init[O](ReadString(txt), WithLexer(lexOpts), SynchronizeAt("option"))
	parseOptions(p)
	_, err := p.Finish()
	if err == nil || err.Error() != `at <input>:2:1: expected the option keyword, got "opton" instead` {
		t.Error("invalid error returned", err)
	}
}

type O struct {
	opts map[string]string
}

func parseOptions(p *Parser[O]) {
	defer p.Synchronize()

	for p.More() {
		p.Expect(OptionToken, "the option keyword")
		p.Expect(StringToken, "an option name, e.g. color")
		p.Expect(StringToken, "an option value, e.g. red")
	}
}

const (
	OptionToken = ScanToken - iota
	StringToken
)

func lexOpts(sc *Scanner) Token {
	switch r := sc.Peek(); r {
	default:
		sc.Advance()
		return Token{Lexeme: string(r)}
	case ' ', '\n', '\t':
		sc.Advance()
		return Ignore
	case 'o':
		sc.Advance()
		rest := "ption"
		for len(rest) > 0 {
			if sc.Advance() != rune(rest[0]) {
				// read the full word to help with error message
				for sc.Peek() != ' ' {
					sc.Advance()
				}
				return Token{}
			}
			rest = rest[1:]
		}
		return Const(OptionToken)
	case '"':
		sc.Advance()
		for sc.Peek() != '"' && sc.Peek() != utf8.RuneError {
			sc.Advance()
		}
		sc.Advance()
		return Auto[string](StringToken, sc)
	}
}
