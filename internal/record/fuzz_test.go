// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package record

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

// FuzzSummarise feeds Summarise arbitrary bytes in place of a passmcp report.
// A report is server-influenced text, so whatever it holds, a summary that
// is accepted must carry only names in the form passmcp gives them and only
// known statuses: no server-chosen string may reach a published table.
func FuzzSummarise(f *testing.F) {
	for _, name := range []string{"stdio-pass.json", "http-oauth-fail.json"} {
		b, err := os.ReadFile(filepath.Join("testdata", name))
		if err != nil {
			f.Fatal(err)
		}
		f.Add(b)
	}
	f.Add([]byte(`{"passmcp":{"schema_version":1,"version":"<script>"},"phases":[{"name":"Net Phase","status":"pass","findings":[{"id":"auth.source.https://evil","status":"fail"},{"id":"x","status":"bogus"}]}]}`))
	f.Add([]byte(`{"passmcp":{"schema_version":1},"auth":{"reached":true,"required":true,"issuer":"i"},"blocked":"yes","phases":[{"name":"net","status":"skip","findings":[{"id":"a","status":"pass"},{"id":"a","status":"warn"},{"id":"a","status":"info"}]}]}`))
	f.Add([]byte(`{"passmcp":{"schema_version":2},"phases":[{}]}`))
	f.Add([]byte(`{"passmcp":{"schema_version":1},"phases":[]}`))
	f.Add([]byte(`not json`))
	f.Fuzz(func(t *testing.T, b []byte) {
		s, err := Summarise(b)
		if err != nil {
			if s != nil {
				t.Fatalf("an error with a summary: %v", err)
			}
			return
		}
		checkSummary(t, s)
	})
}

// checkSummary asserts what a published table relies on.
func checkSummary(t *testing.T, s *Summary) {
	t.Helper()
	checkName(t, "passmcp version", s.PassmcpVersion)
	switch s.AuthScheme {
	case AuthNone, AuthOAuth, AuthOther, AuthUnreached:
	default:
		t.Fatalf("auth scheme %q", s.AuthScheme)
	}
	for phase, status := range s.Phases {
		checkName(t, "phase", phase)
		checkStatus(t, phase, status)
	}
	for id, c := range s.Checks {
		checkName(t, "check", id)
		checkName(t, "phase of "+id, c.Phase)
		checkStatus(t, id, c.Status)
	}
}

// checkName asserts a name is in the form passmcp gives check ids and phase
// names, checked here without Clean's own pattern so the property does not
// assume what it tests: at most 80 bytes of lower-case letters, digits and
// "_.-", starting with a letter or digit, and never a computed
// auth.source.<field> id, whose tail is the server's.
func checkName(t *testing.T, what, name string) {
	t.Helper()
	if !passmcpForm(name) || strings.HasPrefix(name, "auth.source.") {
		t.Fatalf("%s %q reached the summary uncleaned", what, name)
	}
}

func passmcpForm(name string) bool {
	if name == "" || len(name) > 80 || name[0] == '_' || name[0] == '.' || name[0] == '-' {
		return false
	}
	for _, c := range []byte(name) {
		if !idByte(c) {
			return false
		}
	}
	return true
}

func idByte(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_' || c == '.' || c == '-'
}

func checkStatus(t *testing.T, name, status string) {
	t.Helper()
	if _, ok := rank[status]; !ok {
		t.Fatalf("%s has status %q", name, status)
	}
}

// FuzzTail asserts that the bound on kept error text holds for any input,
// and that a cut never splits a rune of valid UTF-8.
func FuzzTail(f *testing.F) {
	f.Add("  short error  ")
	f.Add(string(make([]byte, maxError+10)))
	f.Add("é" + string(make([]byte, maxError-1)) + "日本語")
	f.Fuzz(func(t *testing.T, s string) {
		got := Tail(s)
		if len(got) > maxError+len("…") {
			t.Fatalf("kept %d bytes, the bound is %d", len(got), maxError)
		}
		if utf8.ValidString(s) && !utf8.ValidString(got) {
			t.Fatalf("a cut split a rune: %q", got)
		}
	})
}
