// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package record

import (
	"os"
	"strings"
	"testing"
)

func load(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// stdio-pass.json is a real passmcp 0.0.1 report, trimmed to the fields the
// census reads.
func TestSummariseARealReport(t *testing.T) {
	s, err := Summarise(load(t, "stdio-pass.json"))
	if err != nil {
		t.Fatal(err)
	}
	if s.PassmcpVersion != "0.0.1" || s.AuthScheme != AuthUnreached || s.Blocked {
		t.Fatalf("summary = %+v", s)
	}
	if len(s.Phases) != 5 || s.Phases["discovery"] != "skip" || s.Phases["catalog"] != "pass" {
		t.Fatalf("phases = %v", s.Phases)
	}
	// 26 pass, 10 info and 6 skip findings, each id once.
	if len(s.Checks) != 42 {
		t.Fatalf("%d checks, want 42", len(s.Checks))
	}
	if c := s.Checks["protocol.routing_headers"]; c.Phase != "handshake" || c.Status != "skip" {
		t.Fatalf("a check counts under the phase that ran it: %+v", c)
	}
}

func TestSummariseReducesEachCheckToItsWorstStatus(t *testing.T) {
	s, err := Summarise(load(t, "http-oauth-fail.json"))
	if err != nil {
		t.Fatal(err)
	}
	if s.AuthScheme != AuthOAuth || !s.Blocked {
		t.Fatalf("summary = %+v", s)
	}
	if got := s.Checks["catalog.text.hidden"].Status; got != "fail" {
		t.Fatalf("catalog.text.hidden = %q, want fail", got)
	}
	if _, ok := s.Checks["discovery.prm"]; ok {
		t.Fatal("a finding with an unknown status must not be counted")
	}
	if _, ok := s.Checks["auth.source"]; !ok {
		t.Fatal("a computed family id must be reduced to its family")
	}
	if _, ok := s.Checks[Unrecognised]; !ok {
		t.Fatal("an id not of passmcp's form must be replaced, not published")
	}
	for id := range s.Checks {
		if strings.ContainsAny(id, "<> ") {
			t.Fatalf("server-chosen text reached a check id: %q", id)
		}
	}
}

func TestSummariseRefusesWhatItCannotRead(t *testing.T) {
	for name, body := range map[string]string{
		"not JSON":       "passmcp: no such host",
		"other schema":   `{"passmcp":{"schema_version":2},"phases":[{"name":"net"}]}`,
		"no phases":      `{"passmcp":{"schema_version":1},"phases":[]}`,
		"missing schema": `{"phases":[{"name":"net"}]}`,
	} {
		if _, err := Summarise([]byte(body)); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestAuthScheme(t *testing.T) {
	cases := []struct {
		reached, required, oauth bool
		want                     string
	}{
		{false, true, true, AuthUnreached},
		{true, false, false, AuthNone},
		{true, true, true, AuthOAuth},
		{true, true, false, AuthOther},
	}
	for _, c := range cases {
		if got := authScheme(c.reached, c.required, c.oauth); got != c.want {
			t.Errorf("authScheme(%v, %v, %v) = %q, want %q", c.reached, c.required, c.oauth, got, c.want)
		}
	}
}

func TestClean(t *testing.T) {
	for in, want := range map[string]string{
		"protocol.origin":       "protocol.origin",
		"auth.source.anything":  "auth.source",
		"":                      Unrecognised,
		"UPPER":                 Unrecognised,
		strings.Repeat("a", 81): Unrecognised,
		strings.Repeat("a", 80): strings.Repeat("a", 80),
	} {
		if got := Clean(in); got != want {
			t.Errorf("Clean(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTailKeepsTheEndOnARuneBoundary(t *testing.T) {
	if got := Tail("  short  "); got != "short" {
		t.Fatalf("Tail = %q", got)
	}
	long := strings.Repeat("é", 300) // 600 bytes
	got := Tail(long)
	if !strings.HasPrefix(got, "…") || len(got) > maxError+len("…") {
		t.Fatalf("Tail kept %d bytes", len(got))
	}
	if strings.ContainsRune(got, '�') {
		t.Fatal("Tail split a rune")
	}
}
