ParseKit is a simple, no-surprise library to build parsers and lexers.

It provides the starting blocks needed (and most often forgotten) to make good parser:
 - solid error reporting and parser synchronization
 - efficient buffering while scanning
 - (more to come?)

## Choices made in the package

There are many, many techniques to write a parser ([LALR generators](https://sqlite.org/lemon.html), [PEG](https://www.inf.puc-rio.br/~roberto/lpeg/), [parser combinators](https://serokell.io/blog/parser-combinators-in-haskell), …).

The authors do not claim to have invented anything new, or even smart, but instead chosen a few boring techniques working well together:
 - the program is in control, not using callbacks – leads to a better debugging experience, and code that look more like regular Go
 - the tokenizer uses transition tables instead of regular expressions – we find reading the tables much easier, and code generation ensures an efficient FSA
 - the parser is recursive descent, using panics for stack unwinding and synchronisation – the resulting code is also fairly straightforward, with little verbosity

This choices work well, in practice, to read the kind of files the authors are most often confronted with (configuration files, DHCP leases, SNORT rules, …).

## What next

The test suite is wholly inadequate at the moment, so please feel free to submit bugs and repro cases.
We also need to make the transition table more convenient to use with go generate, instead of a little AWK script.