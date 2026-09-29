// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"satellion.com/passmcp-census/internal/edition"
)

// The test binary stands in for passmcp when CENSUS_FAKE_PASSMCP is set.
func TestMain(m *testing.M) {
	if os.Getenv("CENSUS_FAKE_PASSMCP") == "1" {
		os.Exit(fakePassmcp(os.Args[1:]))
	}
	registryBackoff = time.Millisecond
	os.Exit(m.Run())
}

func fakePassmcp(args []string) int {
	if args[0] == "version" {
		fmt.Println("passmcp 0.0.1")
		return 0
	}
	if strings.HasSuffix(args[1], "/down") {
		fmt.Fprintln(os.Stderr, "dial tcp: connection refused")
		return 1
	}
	fmt.Print(`{"passmcp":{"version":"0.0.1","schema_version":1},"auth":{"reached":true,"required":false},` +
		`"phases":[{"name":"handshake","status":"pass","findings":[{"id":"handshake.server_info","status":"pass"}]},` +
		`{"name":"protocol","status":"fail","findings":[{"id":"protocol.origin","status":"fail"}]}]}`)
	return 2
}

// fakeRegistry lists three servers on loopback, one of them excluded.
func fakeRegistry(t *testing.T) string {
	t.Helper()
	page := `{"servers":[` +
		`{"server":{"name":"io.test/up","version":"1","remotes":[{"type":"streamable-http","url":"https://up.test/mcp"}]}},` +
		`{"server":{"name":"io.test/down","version":"1","remotes":[{"type":"sse","url":"https://down.test/down"}]}},` +
		`{"server":{"name":"io.optout/x","version":"1","remotes":[{"type":"streamable-http","url":"https://x.test/mcp"}]}},` +
		`{"server":{"name":"io.test/keyed","version":"1","remotes":[{"type":"streamable-http","url":"https://k.test/mcp","headers":[{"name":"K","isRequired":true}]}]}}` +
		`],"metadata":{}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(page)) }))
	t.Cleanup(srv.Close)
	return srv.URL
}

type env struct {
	registry, work, out, exclusions string
}

func setup(t *testing.T) env {
	t.Helper()
	t.Setenv("CENSUS_FAKE_PASSMCP", "1")
	dir := t.TempDir()
	e := env{registry: fakeRegistry(t), work: filepath.Join(dir, "work"), out: filepath.Join(dir, "data"), exclusions: filepath.Join(dir, "exclusions.txt")}
	if err := os.WriteFile(e.exclusions, []byte("# test\nio.optout/*\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return e
}

func (e env) runArgs(extra ...string) []string {
	return append([]string{"run", "--edition", "2026-09", "--registry", e.registry, "--passmcp", os.Args[0],
		"--work", e.work, "--exclusions", e.exclusions, "--host-interval", "0s", "--timeout", "30s"}, extra...)
}

func call(args ...string) (int, string, string) {
	var out, errb bytes.Buffer
	code := run(context.Background(), args, &out, &errb)
	return code, out.String(), errb.String()
}

func TestRunThenAggregate(t *testing.T) {
	e := setup(t)
	clock := time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC)
	now = func() time.Time { clock = clock.Add(time.Second); return clock }
	t.Cleanup(func() { now = time.Now })

	code, stdout, stderr := call(e.runArgs()...)
	if code != 0 {
		t.Fatalf("run exited %d\n%s", code, stderr)
	}
	var meta edition.Run
	if err := json.Unmarshal([]byte(stdout), &meta); err != nil {
		t.Fatalf("stdout is not the run's JSON: %v\n%s", err, stdout)
	}
	l := meta.Listing
	if !meta.Complete || meta.PassmcpVersion != "0.0.1" || l.Current != 4 || l.Endpoints != 3 || l.Excluded != 1 ||
		l.Selected != 2 || l.Checked != 2 || l.Skipped["requires headers the user supplies"] != 1 {
		t.Fatalf("run = %+v", meta)
	}
	if meta.Reproduce != "make census EDITION=2026-09 REGISTRY="+e.registry || meta.DurationSeconds <= 0 {
		t.Fatalf("reproduce %q, duration %d", meta.Reproduce, meta.DurationSeconds)
	}
	if !strings.Contains(stderr, "checking 2 endpoints") || strings.Contains(stdout, "checking") {
		t.Fatal("progress belongs on stderr, the result on stdout")
	}

	code, stdout, stderr = call("aggregate", "--edition", "2026-09", "--work", e.work, "--out", e.out)
	if code != 0 {
		t.Fatalf("aggregate exited %d\n%s", code, stderr)
	}
	if !strings.Contains(stdout, `"reports": 1`) {
		t.Fatalf("aggregate stdout:\n%s", stdout)
	}
	checks, err := os.ReadFile(filepath.Join(e.out, "by-check.csv"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(checks), "protocol.origin,protocol,1,0,0,1,0,0,1.0000") {
		t.Fatalf("by-check.csv:\n%s", checks)
	}
	all, _ := os.ReadFile(filepath.Join(e.out, "census.json"))
	if bytes.Contains(all, []byte("up.test")) || bytes.Contains(all, []byte("io.test")) {
		t.Fatal("the published dataset names an endpoint")
	}
}

func TestRunWithALimitIsIncomplete(t *testing.T) {
	e := setup(t)
	code, stdout, _ := call(e.runArgs("--limit", "1")...)
	if code != 1 || !strings.Contains(stdout, `"complete": false`) || !strings.Contains(stdout, `"checked": 1`) {
		t.Fatalf("exit %d\n%s", code, stdout)
	}
}

func TestList(t *testing.T) {
	e := setup(t)
	code, stdout, stderr := call("list", "--edition", "2026-09", "--registry", e.registry, "--work", e.work, "--exclusions", e.exclusions)
	if code != 0 || !strings.Contains(stdout, `"selected": 2`) || !strings.Contains(stdout, `"checked": 0`) {
		t.Fatalf("exit %d\n%s\n%s", code, stdout, stderr)
	}
	if _, err := os.Stat(filepath.Join(e.work, edition.ListingFile)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(e.work, edition.RecordsFile)); err == nil {
		t.Fatal("list must not check anything")
	}
}

func TestUsageErrors(t *testing.T) {
	e := setup(t)
	for _, args := range [][]string{
		{},
		{"nonsense"},
		{"run"},
		{"run", "--edition", "2026-9"},
		{"run", "--edition", "2026-09", "--rps", "10"},
		{"run", "--edition", "2026-09", "--workers", "0"},
		{"run", "--no-such-flag"},
		{"list", "--edition", "x"},
		{"list", "--bad"},
		{"aggregate", "--edition", "x"},
		{"aggregate", "--bad"},
	} {
		if code, _, _ := call(args...); code != 2 {
			t.Errorf("%v exited %d, want 2", args, code)
		}
	}
	_ = e
}

func TestFailures(t *testing.T) {
	e := setup(t)
	missing := filepath.Join(t.TempDir(), "missing")
	cases := map[string][]string{
		"no exclusions file":      {"run", "--edition", "2026-09", "--exclusions", missing, "--work", e.work},
		"no passmcp":              {"run", "--edition", "2026-09", "--exclusions", e.exclusions, "--passmcp", missing, "--work", e.work},
		"list, no exclusions":     {"list", "--edition", "2026-09", "--exclusions", missing, "--work", e.work},
		"list, registry down":     {"list", "--edition", "2026-09", "--exclusions", e.exclusions, "--registry", "http://127.0.0.1:1", "--work", e.work},
		"aggregate, no run":       {"aggregate", "--edition", "2026-09", "--work", missing},
		"run, registry down":      {"run", "--edition", "2026-09", "--exclusions", e.exclusions, "--passmcp", os.Args[0], "--registry", "http://127.0.0.1:1", "--work", e.work},
		"run, work is not a dir":  {"run", "--edition", "2026-09", "--exclusions", e.exclusions, "--passmcp", os.Args[0], "--work", e.exclusions},
		"list, work is not a dir": {"list", "--edition", "2026-09", "--exclusions", e.exclusions, "--work", e.exclusions},
	}
	for name, args := range cases {
		if code, _, stderr := call(args...); code != 1 || !strings.HasPrefix(stderr, "passmcp-census") {
			t.Errorf("%s: exit %d, stderr %q", name, code, stderr)
		}
	}
}

func TestAggregateRefusesInconsistentInput(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	agg := func() int {
		code, _, _ := call("aggregate", "--edition", "2026-09", "--work", dir, "--out", filepath.Join(dir, "out"))
		return code
	}
	write(edition.RunFile, "not json")
	if agg() != 1 {
		t.Fatal("a broken run.json was accepted")
	}
	write(edition.RunFile, `{"edition":"2026-08"}`)
	if agg() != 1 {
		t.Fatal("another edition's run was accepted")
	}
	write(edition.RunFile, `{"edition":"2026-09","listing":{"checked":2}}`)
	if agg() != 1 {
		t.Fatal("a missing records file was accepted")
	}
	write(edition.RecordsFile, `{"outcome":"error"}`+"\n")
	if agg() != 1 {
		t.Fatal("fewer records than the run checked were accepted")
	}
	write(edition.RecordsFile, "{bad\n")
	if agg() != 1 {
		t.Fatal("a broken record was accepted")
	}
	write(edition.RecordsFile, `{"outcome":"error"}`+"\n"+`{"outcome":"timeout"}`+"\n")
	write("out", "a file where the dataset directory goes")
	if agg() != 1 {
		t.Fatal("an unwritable output was not an error")
	}
}

func TestVersionAndHelp(t *testing.T) {
	if code, out, _ := call("version"); code != 0 || out != "passmcp-census dev\n" {
		t.Fatalf("version: %d %q", code, out)
	}
	if code, out, _ := call("help"); code != 0 || !strings.Contains(out, "aggregate") {
		t.Fatalf("help: %d %q", code, out)
	}
}

func TestDefaultsAndReproduce(t *testing.T) {
	fs, o := runFlags(&bytes.Buffer{})
	if err := fs.Parse([]string{"--edition", "2026-09"}); err != nil {
		t.Fatal(err)
	}
	if err := o.validate(); err != nil {
		t.Fatal(err)
	}
	if o.work != filepath.Join("build", "census", "2026-09") || o.registry != DefaultRegistry || o.rps != 1 {
		t.Fatalf("defaults = %+v", o)
	}
	if got := reproduce(o); got != "make census EDITION=2026-09" {
		t.Fatalf("reproduce = %q", got)
	}
	if !strings.HasPrefix(userAgent(), "passmcp-census/dev (+https://github.com/sebastienrousseau/passmcp-census)") {
		t.Fatalf("User-Agent = %q", userAgent())
	}
}
