// SPDX:Apache-2.0
package cmd

import (
	"fmt"
	"os"
	"strings"
)

func splitFileArgs(filename string) ([]string, error) {
	data_bin, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	data_str := string(data_bin)
	ret := make([]string, 0)
	buff := strings.Builder{}
	mode := 0
	esc := ' '
	for _, c := range data_str {
		switch mode {
		case 0:
			// Start of arg search
			if c == ' ' || c == '\n' || c == '\r' {
				// Keep searching
				continue
			} else {
				mode = 1
			}
			fallthrough
		case 1:
			// Inside an argument
			if c == ' ' || c == '\n' || c == '\r' {
				// End of argument.
				ret = append(ret, buff.String())
				buff.Reset()
				mode = 0
			} else if c == '\'' || c == '"' {
				esc = c
				mode = 2
			} else if c == '\\' {
				mode = 4
			} else {
				if _, err := buff.WriteRune(c); err != nil {
					return nil, err
				}
			}
		case 2:
			// Quoted partial string.
			// Ending the quote does not end the argument.
			if c == esc {
				mode = 0
			} else if c == '\\' {
				mode = 2
			} else {
				if _, err := buff.WriteRune(c); err != nil {
					return nil, err
				}
			}
		case 3:
			// Escape inside quote
			// Note: no special unicode handling.
			if _, err := buff.WriteRune(c); err != nil {
				return nil, err
			}
			mode = 2
		case 4:
			// Escape outside quote
			// Note: no special unicode handling.
			if _, err := buff.WriteRune(c); err != nil {
				return nil, err
			}
			mode = 1
		}
	}
	if mode > 0 {
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
