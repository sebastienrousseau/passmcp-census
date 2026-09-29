// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// fakeRegistry serves the two fixture pages on loopback: no test here
// contacts a real registry.
func fakeRegistry(t *testing.T) (*httptest.Server, *[]string) {
	t.Helper()
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.URL.RequestURI()+" ua="+r.UserAgent())
		if r.URL.Path != "/v0/servers" {
			http.NotFound(w, r)
			return
		}
		name := "testdata/page1.json"
		if r.URL.Query().Get("cursor") == "page2" {
			name = "testdata/page2.json"
		}
		b, err := os.ReadFile(name)
		if err != nil {
			t.Error(err)
		}
		_, _ = w.Write(b)
	}))
	t.Cleanup(srv.Close)
	return srv, &seen
}

func TestListFollowsTheMethod(t *testing.T) {
	srv, seen := fakeRegistry(t)
	var progress strings.Builder
	c := &Client{Base: srv.URL + "/", UserAgent: "census-test", PageSize: 7, Progress: &progress}
	l, err := c.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(progress.String(), "page 2, 9 entries so far") {
		t.Fatalf("progress:\n%s", progress.String())
	}
	if len(*seen) != 2 || !strings.Contains((*seen)[0], "limit=7") || !strings.Contains((*seen)[0], "version=latest") || !strings.Contains((*seen)[1], "cursor=page2") ||
		!strings.HasSuffix((*seen)[0], "ua=census-test") {
		t.Fatalf("requests = %v", *seen)
	}
	// Nine named entries across the two pages; "not an object" and the
	// nameless one are not entries.
	if l.Entries != 9 || l.Current != 6 {
		t.Fatalf("entries %d current %d", l.Entries, l.Current)
	}
	var urls []string
	for _, e := range l.Endpoints {
		urls = append(urls, e.URL)
	}
	want := "https://mcp.example.com/mcp https://mcp.example.com/sse https://keyed.example.com/open https://flat.example.com/mcp https://mcp.example.com/mcp"
	if got := strings.Join(urls, " "); got != want {
		t.Fatalf("endpoints\n got %s\nwant %s", got, want)
	}
	if l.Endpoints[3].Transport != "streamable-http" {
		t.Fatalf("transport_type is read: %+v", l.Endpoints[3])
	}
	reasons := map[string]int{}
	for _, s := range l.Skipped {
		reasons[s.Reason]++
	}
	wantReasons := map[string]int{ReasonTemplate: 1, ReasonHeaders: 1, ReasonTransport: 1, ReasonNotHTTP: 1, ReasonCredentials: 1}
	for r, n := range wantReasons {
		if reasons[r] != n {
			t.Errorf("skipped %q: %d, want %d (all: %v)", r, reasons[r], n, reasons)
		}
	}
}

func TestDistinctKeepsTheFirstListingOfAURL(t *testing.T) {
	eps := []Endpoint{{Name: "a", URL: "u1"}, {Name: "b", URL: "u2"}, {Name: "c", URL: "u1"}}
	got, dup := Distinct(eps)
	if dup != 1 || len(got) != 2 || got[0].Name != "a" || got[1].Name != "b" {
		t.Fatalf("Distinct = %v, %d", got, dup)
	}
}

func TestListRetriesATransientFailure(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		switch calls.Add(1) {
		case 1:
			w.WriteHeader(http.StatusInternalServerError)
		case 2:
			w.WriteHeader(http.StatusTooManyRequests)
		default:
			_, _ = w.Write([]byte(`{"servers":[],"metadata":{}}`))
		}
	}))
	defer srv.Close()
	c := &Client{Base: srv.URL, Retries: 2, Backoff: time.Millisecond}
	if _, err := c.List(context.Background()); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 3 {
		t.Fatalf("%d calls, want 3", calls.Load())
	}
}

func TestListGivesUp(t *testing.T) {
	status := func(code int) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(code) }))
	}
	for _, tc := range []struct {
		code    int
		retries int
		calls   string
	}{{http.StatusBadGateway, 1, "after its retries"}, {http.StatusNotFound, 3, "at once on a 4xx"}} {
		srv := status(tc.code)
		c := &Client{Base: srv.URL, Retries: tc.retries, Backoff: time.Millisecond}
		_, err := c.List(context.Background())
		srv.Close()
		if err == nil || !strings.Contains(err.Error(), "HTTP") {
			t.Errorf("HTTP %d: err = %v, want a failure %s", tc.code, err, tc.calls)
		}
	}
}

func TestListStopsWhenTheContextEndsDuringABackoff(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusServiceUnavailable) }))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	c := &Client{Base: srv.URL, Retries: 5, Backoff: time.Hour}
	if _, err := c.List(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v", err)
	}
}

func TestListRefusesWhatIsNotAPage(t *testing.T) {
	for name, body := range map[string]string{
		"not JSON":  "<html>",
		"too large": `{"servers":[],"x":"` + strings.Repeat("a", maxPageBody) + `"}`,
	} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(body)) }))
		_, err := (&Client{Base: srv.URL}).List(context.Background())
		srv.Close()
		if err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestListRefusesARegistryThatNeverStopsPaging(t *testing.T) {
	var n atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"servers":[],"metadata":{"nextCursor":"c` + strings.Repeat("x", int(n.Add(1)%3)) + `"}}`))
	}))
	defer srv.Close()
	if _, err := (&Client{Base: srv.URL}).List(context.Background()); err == nil || !strings.Contains(err.Error(), "pages") {
		t.Fatalf("err = %v", err)
	}
}

func TestListStopsOnARepeatedCursor(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"servers":[],"metadata":{"nextCursor":"same"}}`))
	}))
	defer srv.Close()
	if _, err := (&Client{Base: srv.URL}).List(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestListRefusesABadBaseOrAnUnreachableRegistry(t *testing.T) {
	if _, err := (&Client{Base: "not a url"}).List(context.Background()); err == nil {
		t.Fatal("a relative base was accepted")
	}
	srv := httptest.NewServer(http.NotFoundHandler())
	base := srv.URL
	srv.Close()
	if _, err := (&Client{Base: base}).List(context.Background()); err == nil {
		t.Fatal("an unreachable registry was not an error")
	}
	if _, err := (&Client{Base: "http://127.0.0.1:1\x7f"}).List(context.Background()); err == nil {
		t.Fatal("a URL with a control character was accepted")
	}
}
