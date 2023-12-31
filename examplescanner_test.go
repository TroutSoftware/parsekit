package parsekit_test

import (
	"fmt"
	"net/netip"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/TroutSoftware/parsekit"
)

func Example() {
	p := parsekit.Init[Lease](
		parsekit.ReadFiles("testdata/example_dhcp1"),
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

	var err error

	p.Expect(IdentToken, "lease")
	p.Expect('{', "opening bracket")
	for p.More() {
		if p.Match('}') {
			return
		}

		p.Expect(IdentToken, "option")
		opt := p.Lit
		switch opt {
		case "interface":
			p.Expect(StringToken, "interface")
			p.Value.Interface, err = strconv.Unquote(p.Lit)
			if err != nil {
				p.Errf("invalid interface name %q: %s", p.Lit, err)
			}
			p.Expect(';', ";")
		case "fixed-address":
			p.Expect(IPToken, "IP address")
			p.Value.FixedAddress, err = netip.ParseAddr(p.Lit)
			if err != nil {
				p.Errf("invalid IP address %q: %s", p.Lit, err)
			}
			p.Expect(';', ";")
		case "expire":
			p.Expect(NumberToken, "number")
			p.Expect(DateTimeToken, "date and time of expiration")
			p.Value.Expire, err = time.Parse("2006/01/02 15:04:05", p.Lit)
			if err != nil {
				p.Errf("invalid time of expiration %q: %s", p.Lit, err)
			}
			p.Expect(';', ";")
		default:
			for !p.Match(';') {
				p.Skip()
			}
		}
	}
}

const (
	NumberToken rune = -1 - iota
	IPToken
	DateTimeToken
	IdentToken
	StringToken
	InvalidType
)

func scantk(sc *parsekit.Scanner, tk rune) (rune, int) {
	switch {
	case tk == 0:
		return 0, 0
	case tk == '{', tk == '}', tk == ';':
		return tk, 1
	case tk == '"':
		return StringToken, sc.LexString()
	case '0' <= tk && tk <= '9':
		return scanumeral(sc, tk)
	default:
		return IdentToken, sc.LexIdent()
	}
}

// unrolled state machine
// transitions generated by calling transgen.awk
// TODO make this go:generatable
func scanumeral(sc *parsekit.Scanner, lead rune) (rune, int) {
	const (
		numeral uint8 = iota
		date
		time
		ip

		final
	)

	var transitions = [final][]byte{
		numeral: {' ', final, ';', final, '.', ip, '/', date, '0', numeral, '1', numeral, '2', numeral, '3', numeral, '4', numeral, '5', numeral, '6', numeral, '7', numeral, '8', numeral, '9', numeral},
		ip:      {':', ip, ' ', final, ';', final, '.', ip, '0', ip, '1', ip, '2', ip, '3', ip, '4', ip, '5', ip, '6', ip, '7', ip, '8', ip, '9', ip},
		time:    {':', time, ';', final, '0', time, '1', time, '2', time, '3', time, '4', time, '5', time, '6', time, '7', time, '8', time, '9', time},
		date:    {' ', time, '/', date, '0', date, '1', date, '2', date, '3', date, '4', date, '5', date, '6', date, '7', date, '8', date, '9', date},
	}

	st, n := sc.ScanWithTable(transitions[:])
	switch st {
	default:
		return InvalidType, n
	case numeral:
		return NumberToken, n
	case time:
		return DateTimeToken, n
	case ip:
		return IPToken, n + utf8.RuneLen(lead) - 1 // drop space or training ;
	}
}
