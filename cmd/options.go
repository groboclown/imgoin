// SPDX:Apache-2.0
package cmd

import (
	"fmt"
	"os"
	"strings"
)

type splitMode int

const (
	// Ordering matters, so no iota use.
	smArgSearch splitMode = 0
	smComment   splitMode = 1
	smNormal    splitMode = 2
	smQuote     splitMode = 3
	smEscQuote  splitMode = 4
	smEscape    splitMode = 5

	// Marker for the first mode for parsing within an argument.
	smInArgument = smNormal
)

func SplitFileArgs(filename string) ([]string, error) {
	data_bin, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return SplitArgs(string(data_bin))
}

func SplitArgs(data_str string) ([]string, error) {
	ret := make([]string, 0)
	buff := strings.Builder{}
	mode := smArgSearch
	quot := ' '
	for _, c := range data_str {
		switch mode {
		case smArgSearch:
			// Start of arg search
			switch c {
			case ' ', '\t', '\n', '\r':
				// Keep searching
			case '#':
				// Comment start
				mode = smComment
			case '\'', '"':
				// Start an argument, with quoted mode.
				quot = c
				mode = smQuote
			case '\\':
				// Start an argument, begin with an escaped character.
				mode = smEscape
			default:
				// Start an argument, normal mode.
				mode = smNormal
				if _, err := buff.WriteRune(c); err != nil {
					return nil, err
				}
			}
		case smComment:
			// Inside a newline-terminating comment.
			// Only care about newlines.
			switch c {
			case '\n', '\r':
				mode = smArgSearch
			}
		case smNormal:
			// Inside an argument with normal parsing characteristics.
			switch c {
			case ' ', '\t', '\n', '\r':
				// End of argument, switch to normal start arg search.
				ret = append(ret, buff.String())
				buff.Reset()
				mode = smArgSearch
			case '#':
				// End of argument, then start a comment.
				ret = append(ret, buff.String())
				buff.Reset()
				mode = smComment
			case '\'', '"':
				// Switch to quoted mode.
				quot = c
				mode = smQuote
			case '\\':
				// Read an escaped character.
				mode = smEscape
			default:
				// Normal argument character
				if _, err := buff.WriteRune(c); err != nil {
					return nil, err
				}
			}
		case smQuote:
			// Quoted partial string.
			// Ending the quote does not end the argument.
			switch c {
			case quot:
				// End quote.  This doesn't go back to argument search,
				// but instead goes back to in-argument reading.
				// This allows for the form `abc" "def`.
				mode = smNormal
			case '\\':
				// Escape the next character inside the quote.
				mode = smEscQuote
			default:
				// Quoted character read.
				if _, err := buff.WriteRune(c); err != nil {
					return nil, err
				}
			}
		case smEscQuote:
			// Escape inside quote
			// Note: no special unicode handling.
			if _, err := buff.WriteRune(c); err != nil {
				return nil, err
			}
			mode = smQuote
		case smEscape:
			// Escape outside quote
			// Note: no special unicode handling.
			if _, err := buff.WriteRune(c); err != nil {
				return nil, err
			}
			mode = smNormal
		}
	}
	if mode >= smInArgument {
		ret = append(ret, buff.String())
	}
	return ret, nil
}

func parseBool(value string) (bool, error) {
	if ret, ok := boolValues[value]; ok {
		return ret, nil
	}
	return false, fmt.Errorf("Error: value must be true or false, found '%s'", value)
}

var boolValues map[string]bool = map[string]bool{
	"true":  true,
	"false": false,
	"t":     true,
	"f":     false,
	"yes":   true,
	"no":    false,
	"y":     true,
	"n":     false,
	"on":    true,
	"off":   false,
	"1":     true,
	"0":     false,
}
