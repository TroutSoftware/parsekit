# transition table generator
#   generate an efficient table to record all transitions in a state machine
#  input: a set of transitions expressed as <state> <character> -> <state>
#  output: Go code about said transition

BEGIN    { 
    for (i = 0; i <= 127; i++) {
        t = sprintf("%c", i)
        _ord_[t] = i
    }
}

NF == 0 { next } # skip empty lines

$2 ~ /[a-zA-Z0-9]…[a-zA-Z0-9]/ { # handle ranges
	states[$1] = $1
	split($2, parts, "…")
	for (p = _ord_[parts[1]]; p <= _ord_[parts[2]]; p++) {
		pc = sprintf("%c", p)
		chars[pc] = pc
		transitions[$1, pc] = $4
	}
	next
 }

 $2 == "CA" { states[$1] = $1; catchalls[$1] = $4 ; next } 

 { if ($2 == "SP") $2 = " "
   if ($2 == "\\") $2 = "\\\\"
    states[$1] = $1; chars[$2] = $2; transitions[$1, $2] = $4 }

END {
	iota = 0
	print "const ("
	for (st in states) {
		print st " = " iota
		iota++
	}
	print "final = " iota
	print ")"
	print ""
	print "var transitions = [final][]byte{"
	for (st in states) {
		printf "\t" st ": {"
		if (st in catchalls)
			printf "'\\x00', " catchalls[st] ", "
		for (ch in chars) {
			if ((st, ch) in transitions)
				printf "'" ch "', " transitions[st, ch] ", "
		}
		print "},"
	}
	print "}"
}