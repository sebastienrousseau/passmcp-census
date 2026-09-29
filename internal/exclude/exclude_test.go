// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package exclude

import (
	"errors"
	"strings"
	"testing"
	"testing/iotest"
)

func TestExcludes(t *testing.T) {
	l, err := Parse(strings.NewReader("# comment\n\n io.example/one \nio.other/*\n"))
	if err != nil {
		t.Fatal(err)
	}
	if l.Len() != 2 {
		t.Fatalf("Len = %d", l.Len())
	}
	for name, want := range map[string]bool{
		"io.example/one":  true,
		"io.example/two":  false,
		"io.other/x":      true,
		"io.other":        false,
		"io.otherwise/x":  false,
		"# comment":       false,
		"io.example/one2": false,
	} {
		if got := l.Excludes(name); got != want {
			t.Errorf("Excludes(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestNilListExcludesNothing(t *testing.T) {
	var l *List
	if l.Excludes("x") || l.Len() != 0 {
		t.Fatal("a nil list excluded something")
	}
}

func TestParseReportsAReadError(t *testing.T) {
	boom := errors.New("boom")
	if _, err := Parse(iotest.ErrReader(boom)); !errors.Is(err, boom) {
		t.Fatalf("err = %v", err)
	}
}
