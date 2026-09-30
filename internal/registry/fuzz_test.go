// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"bytes"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// FuzzListing feeds one registry page of arbitrary bytes through the
// selection. The registry is untrusted input: whatever a page holds, every
// endpoint selected must be one the census may check as listed (a remote
// HTTP transport, an absolute http(s) URL with no template and no
// credentials), every endpoint set aside must carry a known reason, and
// Distinct must leave each URL once.
func FuzzListing(f *testing.F) {
	for _, name := range []string{"page1.json", "page2.json"} {
		b, err := os.ReadFile(filepath.Join("testdata", name))
		if err != nil {
			f.Fatal(err)
		}
		f.Add(b)
	}
	f.Add([]byte(`{"servers":[{"name":"a","remotes":[{"transport_type":"sse","url":"http://u:p@h/"},{"type":"sse","url":"ftp://h/"},{"type":"stdio"}]}]}`))
	f.Add([]byte(`{"servers":[{"name":"a","_meta":{"` + officialKey + `":{"status":"active","isLatest":null}},"remotes":[{"type":"sse","url":"https://h/x"},{"type":"sse","url":"https://h/x"}]}]}`))
	f.Add([]byte(`{"servers":[{"server":null,"_meta":{"` + officialKey + `":"not an object"}}]}`))
	f.Add([]byte(`{"servers":"nope"}`))
	f.Fuzz(func(t *testing.T, b []byte) {
		p, err := decodePage(bytes.NewReader(b))
		if err != nil {
			return
		}
		var l Listing
		for _, raw := range p.Servers {
			l.add(raw)
		}
		checkListing(t, l)
		checkDistinct(t, l.Endpoints)
	})
}

func checkListing(t *testing.T, l Listing) {
	t.Helper()
	if l.Current > l.Entries || l.Entries < 0 {
		t.Fatalf("%d current of %d entries", l.Current, l.Entries)
	}
	for _, e := range l.Endpoints {
		checkCheckable(t, e)
	}
	known := map[string]bool{ReasonTransport: true, ReasonTemplate: true, ReasonNotHTTP: true, ReasonCredentials: true, ReasonHeaders: true}
	for _, s := range l.Skipped {
		if !known[s.Reason] {
			t.Fatalf("%q set aside for %q", s.URL, s.Reason)
		}
	}
}

// checkCheckable asserts, independently of urlReason, that an endpoint can
// be connected to as listed.
func checkCheckable(t *testing.T, e Endpoint) {
	t.Helper()
	if e.Transport != "streamable-http" && e.Transport != "sse" {
		t.Fatalf("selected transport %q", e.Transport)
	}
	if e.Name == "" {
		t.Fatalf("selected an endpoint of a nameless entry: %q", e.URL)
	}
	if strings.ContainsAny(e.URL, "{}") {
		t.Fatalf("selected a templated URL %q", e.URL)
	}
	u, err := url.Parse(e.URL)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil {
		t.Fatalf("selected URL %q is not a bare absolute http(s) URL", e.URL)
	}
}

// checkDistinct asserts Distinct keeps the first of each URL, in order, and
// counts the rest.
func checkDistinct(t *testing.T, eps []Endpoint) {
	t.Helper()
	out, dropped := Distinct(eps)
	if len(out)+dropped != len(eps) {
		t.Fatalf("%d kept and %d dropped of %d", len(out), dropped, len(eps))
	}
	seen := map[string]bool{}
	i := 0
	for _, e := range eps {
		if seen[e.URL] {
			continue
		}
		seen[e.URL] = true
		if i >= len(out) || out[i] != e {
			t.Fatalf("at %d of %v, want the first listing %+v", i, out, e)
		}
		i++
	}
}
