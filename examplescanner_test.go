package parsekit_test

import (
	"fmt"
	"net/netip"
	"time"
	"unicode/utf8"

	"github.com/TroutSoftware/parsekit/v2"
)

func Example() {
	p := parsekit.Init[Lease](
		parsekit.ReadFile("testdata/example_dhcp1"),
		parsekit.WithLexer(scantk),
		parsekit.SynchronizeAt("lease"),
	)

	ParseLease(p)
	lease, err := p.Finish()
	if err != nil {
		fmt.Printf("cannot parse lease file: %s", err)
		return
	}

	fmt.Println(lease)
	// Output: {eth0 10.67.21.85 2023-11-03 11:27:26 +0000 UTC}
}

type Lease struct {
	Interface    string
	FixedAddress netip.Addr
	Expire       time.Time
}

func ParseLease(p *parsekit.Parser[Lease]) {
	defer p.Synchronize()

	p.Expect(IdentToken, "lease")
	p.Expect('{', "opening bracket")
	for p.More() {
		if p.Match('}') {
			return
		}

		p.Expect(IdentToken, "option")
		switch p.Lit() {
		case "interface":
			p.Expect(StringToken, "interface")
			p.Value.Interface = p.Val().(string)
			p.Expect(';', ";")
		case "fixed-address":
			p.Expect(IPToken, "IP address")
			p.Value.FixedAddress = p.Val().(netip.Addr)
			p.Expect(';', ";")
		case "expire":
			p.Expect(NumberToken, "number")
			p.Expect(DateTimeToken, "date and time of expiration")
			p.Value.Expire = time.Time(p.Val().(LTime))
			p.Expect(';', ";")
		default:
			for !p.Match(';') {
				p.Skip()
			}
		}
	}
}

type LTime time.Time

func (t *LTime) UnmarshalText(dt []byte) error {
	u, err := time.Parse("2006/01/02 15:04:05", string(dt))
	if err != nil {
		return err
	}
	*t = (LTime)(u)
	return nil
}

const (
	NumberToken rune = -1 - iota
	IPToken
	DateTimeToken
	IdentToken
	StringToken
	InvalidType
)

func scantk(sc *parsekit.Scanner) parsekit.Token {
	switch tk := sc.Advance(); {
	case tk == ' ':
		return parsekit.Ignore // empty space

	case tk == '{', tk == '}', tk == ';':
		return parsekit.Const(tk)

	case tk == '"':
		for sc.Peek() != '"' && sc.Peek() != utf8.RuneError {
			sc.Advance()
		}
		if sc.Peek() == utf8.RuneError {
			return parsekit.EOF
		}
		sc.Advance() // terminating '"'
		return parsekit.Auto[string](StringToken, sc)

	case '0' <= tk && tk <= '9':
		guess := NumberToken
		for {
			if sc.Peek() >= '0' && sc.Peek() <= '9' {
				sc.Advance()
			} else if sc.Peek() == '/' {
				guess = DateTimeToken
				sc.Advance()
			} else if sc.Peek() == '.' {
				guess = IPToken
				sc.Advance()
			} else if (sc.Peek() == ' ' || sc.Peek() == ':') && guess == DateTimeToken {
				sc.Advance()
			} else {
				break
			}
		}
		switch guess {
		case DateTimeToken:
			return parsekit.Auto[LTime](guess, sc)
		case IPToken:
			return parsekit.Auto[netip.Addr](guess, sc)
		default:
			return parsekit.Auto[int](guess, sc)
		}

	case 'a' <= tk && tk <= 'z' || tk == '-':
		for 'a' <= sc.Peek() && sc.Peek() <= 'z' || sc.Peek() == '-' {
			sc.Advance()
		}
		return parsekit.Const(IdentToken)
	}

	return parsekit.EOF
}
