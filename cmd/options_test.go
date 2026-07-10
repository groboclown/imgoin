// SPDX:Apache-2.0
package cmd_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/groboclown/imgoin/cmd"
)

type sfaTD struct {
	inp string
	exp []string
}

func Test_SplitArgs(t *testing.T) {
	testData := []sfaTD{
		{inp: "", exp: []string{}},
		{inp: "\\", exp: []string{""}},
		{inp: "'", exp: []string{""}},
		{inp: "\"\\", exp: []string{""}},
		{inp: "'''", exp: []string{""}},
		{inp: "a", exp: []string{"a"}},
		{inp: "abc", exp: []string{"abc"}},
		{inp: "abc ", exp: []string{"abc"}},
		{inp: " abc ", exp: []string{"abc"}},
		{inp: "a b", exp: []string{"a", "b"}},
		{inp: "a\nb", exp: []string{"a", "b"}},
		{inp: "a\rb", exp: []string{"a", "b"}},
		{inp: "a\r\nb", exp: []string{"a", "b"}},
		{inp: "a\r\n  \r\n\r\nb", exp: []string{"a", "b"}},
		{inp: "# foo", exp: []string{}},
		{inp: "a\n# foo \nb", exp: []string{"a", "b"}},
		{inp: "a b# foo", exp: []string{"a", "b"}},
		{inp: "a b# foo\n ", exp: []string{"a", "b"}},
		{inp: "'# foo'", exp: []string{"# foo"}},
		{inp: "a'12\"34'\"c\"d", exp: []string{"a12\"34cd"}},
		{inp: "\\\\", exp: []string{"\\"}},
		{inp: "\\a", exp: []string{"a"}},
		{inp: "\"\\\"\"", exp: []string{"\""}},
		{inp: "'\\''", exp: []string{"'"}},
		{inp: "\"a b\"", exp: []string{"a b"}},
	}

	for _, data := range testData {
		act, err := cmd.SplitArgs(data.inp)
		if err != nil {
			t.Errorf("Failed to create: %v", err)
		} else if diff := cmp.Diff(data.exp, act); diff != "" {
			t.Error(diff)
		}
	}
}
