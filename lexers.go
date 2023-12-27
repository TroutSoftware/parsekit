package parsekit

// LexString returns the number of characters in the next string value.
// Strings are delimited between double quotes, single quotes and backtick, and support \" escaping.
// This lexer assumes the first character of the string (the initial ") has not yet been consumed.
func (s *Scanner) LexString() (n int) {
	w := s.br.window()
	// sanity check
	if len(w) == 0 || !quotechars[w[0]] {
		return 0
	}

	offset := 1
	escaped := false
	quote := w[0]

winLoop:
	for _, char := range w[offset:] {
		offset++
		switch {
		case escaped:
			escaped = false
		case char == quote:
			return offset
		case char == '\\':
			escaped = true
		}
	}

	if s.br.extend() == 0 {
		return offset
	}
	w = s.br.window()
	goto winLoop
}

var quotechars = [256]bool{'"': true, '\'': true, '`': true}

// LexIdent returns the number of characters in the next identifier value.
// Identifier are recognized as characters (a-zA-Z ASCII) and underscore or dash.
func (s *Scanner) LexIdent() int {
	w := s.br.window()
	offset := 0
winLoop:
	for _, c := range w[offset:] {
		if !identchars[c] {
			return offset
		}
		offset++
	}
	if s.br.extend() == 0 {
		return offset
	}
	w = s.br.window()
	goto winLoop
}

var identchars = [256]bool{
	'a': true, 'b': true, 'c': true, 'd': true, 'e': true, 'f': true, 'g': true, 'h': true, 'i': true, 'j': true, 'k': true, 'l': true, 'm': true, 'n': true, 'o': true, 'p': true, 'q': true, 'r': true, 's': true, 't': true, 'u': true, 'v': true, 'w': true, 'x': true, 'y': true, 'z': true,
	'A': true, 'B': true, 'C': true, 'D': true, 'E': true, 'F': true, 'G': true, 'H': true, 'I': true, 'J': true, 'K': true, 'L': true, 'M': true, 'N': true, 'O': true, 'P': true, 'Q': true, 'R': true, 'S': true, 'T': true, 'U': true, 'V': true, 'W': true, 'X': true, 'Y': true,
	'_': true, '-': true,
}
