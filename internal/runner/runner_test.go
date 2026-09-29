// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package runner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"satellion.com/passmcp-census/internal/limiter"
	"satellion.com/passmcp-census/internal/record"
	"satellion.com/passmcp-census/internal/registry"
)

// The test binary stands in for passmcp when CENSUS_FAKE_PASSMCP is set, so
// these tests run on every OS and contact nothing. What it does depends on
// the endpoint's path.
func TestMain(m *testing.M) {
	switch os.Getenv("CENSUS_FAKE_PASSMCP") {
	case "1":
		os.Exit(fakePassmcp(os.Args[1:]))
	case "broken":
		fmt.Fprintln(os.Stderr, "passmcp: broken")
		os.Exit(3)
	}
	os.Exit(m.Run())
}

const fakeReport = `{"passmcp":{"version":"0.0.1","schema_version":1},"auth":{"reached":true},` +
	`"phases":[{"name":"net","status":"pass","findings":[{"id":"net.dns","status":"pass"},{"id":"net.tls","status":"%s"}]}]}`

func fakePassmcp(args []string) int {
	if len(args) == 1 && args[0] == "version" {
		fmt.Println("passmcp 0.0.1")
		return 0
	}
	if log := os.Getenv("CENSUS_FAKE_LOG"); log != "" {
		cfg, _ := os.ReadFile(os.Getenv("PASSMCP_CONFIG"))
		line := strings.Join(args, " ") + "\ntoken=" + os.Getenv("PASSMCP_TOKEN") + "\nconfig=" + strings.TrimSpace(string(cfg)) + "\n"
		_ = os.WriteFile(log, []byte(line), 0o600)
	}
	switch {
	case strings.HasSuffix(args[1], "/pass"):
		fmt.Printf(fakeReport, "pass")
		return 0
	case strings.HasSuffix(args[1], "/fail"):
		fmt.Printf(fakeReport, "fail")
		return 2
	case strings.HasSuffix(args[1], "/garbage"):
		fmt.Print("not a report")
		return 0
	case strings.HasSuffix(args[1], "/hang"):
		time.Sleep(time.Minute)
		return 0
	default:
		fmt.Fprintln(os.Stderr, "passmcp: no such host")
		return 1
	}
}

func fake(t *testing.T) *Passmcp {
	t.Helper()
	t.Setenv("CENSUS_FAKE_PASSMCP", "1")
	return &Passmcp{Bin: os.Args[0], UserAgent: "passmcp-census/test", RPS: 1, CallTimeout: 20 * time.Second}
}

func TestArgsAreTheMethod(t *testing.T) {
	p := &Passmcp{RPS: 1, CallTimeout: 20 * time.Second, UserAgent: "ua"}
	got := strings.Join(p.Args("https://x.example/mcp"), " ")
	want := "check https://x.example/mcp --phases net,discovery,handshake,protocol,catalog --auth none --output json" +
		" --no-color --log-level error --rps 1 --timeout 20s --user-agent ua"
	if got != want {
		t.Fatalf("args\n got %s\nwant %s", got, want)
	}
	for _, invoking := range []string{"execution", "performance", "resilience", "auth"} {
		if slices.Contains(Phases, invoking) {
			t.Fatalf("the census must not run the %s phase", invoking)
		}
	}
	if strings.Contains(strings.Join((&Passmcp{}).Args("u"), " "), "--user-agent") {
		t.Fatal("an empty User-Agent must not be passed")
	}
}

func TestCheckRunsWithoutTheOperatorsCredentials(t *testing.T) {
	p := fake(t)
	log := filepath.Join(t.TempDir(), "log")
	t.Setenv("CENSUS_FAKE_LOG", log)
	t.Setenv("PASSMCP_TOKEN", "operator-secret")
	t.Setenv("passmcp_client_secret", "lower-case-too")
	rec := p.Check(context.Background(), registry.Endpoint{Name: "n", URL: "https://a.example/pass", Transport: "sse"})
	if rec.Outcome != record.OutcomeReport || rec.Summary == nil || rec.Transport != "sse" || rec.Name != "n" {
		t.Fatalf("record = %+v", rec)
	}
	b, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "token=\n") || !strings.Contains(string(b), "config={}") {
		t.Fatalf("passmcp saw:\n%s", b)
	}
	for _, kv := range Environ([]string{"PASSMCP_TOKEN=x", "passmcp_client_secret=y", "HOME=/h"}, "/c") {
		if strings.Contains(kv, "=x") || strings.Contains(kv, "=y") {
			t.Fatalf("Environ kept %q", kv)
		}
	}
}

func TestCheckOutcomes(t *testing.T) {
	p := fake(t)
	cases := map[string]string{
		"/pass":    record.OutcomeReport,
		"/fail":    record.OutcomeReport,
		"/garbage": record.OutcomeError,
		"/crash":   record.OutcomeError,
	}
	for path, want := range cases {
		rec := p.Check(context.Background(), registry.Endpoint{URL: "https://a.example" + path})
		if rec.Outcome != want {
			t.Errorf("%s: outcome %q (%s), want %q", path, rec.Outcome, rec.Error, want)
		}
	}
	rec := p.Check(context.Background(), registry.Endpoint{URL: "https://a.example/fail"})
	if rec.Exit != 2 || rec.Checks["net.tls"].Status != "fail" {
		t.Fatalf("exit 2 carries a report: %+v", rec)
	}
	rec = p.Check(context.Background(), registry.Endpoint{URL: "https://a.example/crash"})
	if rec.Exit != 1 || !strings.Contains(rec.Error, "no such host") {
		t.Fatalf("crash: %+v", rec)
	}
}

func TestCheckTimesOut(t *testing.T) {
	p := fake(t)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	rec := p.Check(ctx, registry.Endpoint{URL: "https://a.example/hang"})
	if rec.Outcome != record.OutcomeTimeout {
		t.Fatalf("outcome %q, want timeout", rec.Outcome)
	}
}

func TestCheckWithoutABinary(t *testing.T) {
	p := &Passmcp{Bin: filepath.Join(t.TempDir(), "no-passmcp")}
	rec := p.Check(context.Background(), registry.Endpoint{URL: "https://a.example/pass"})
	if rec.Outcome != record.OutcomeError || rec.Exit != -1 {
		t.Fatalf("record = %+v", rec)
	}
	if _, err := p.Version(context.Background()); err == nil {
		t.Fatal("Version succeeded without a binary")
	}
}

func TestVersion(t *testing.T) {
	v, err := fake(t).Version(context.Background())
	if err != nil || v != "0.0.1" {
		t.Fatalf("Version = %q, %v", v, err)
	}
	t.Setenv("CENSUS_FAKE_PASSMCP", "broken")
	p := &Passmcp{Bin: os.Args[0]}
	if _, err := p.Version(context.Background()); err == nil || !strings.Contains(err.Error(), "exited 3") {
		t.Fatal("a failing version command was not an error")
	}
}

// stub is a Checker that records the order of checks and when they ran.
type stub struct {
	mu      sync.Mutex
	started []string
	delay   time.Duration
}

func (s *stub) Check(ctx context.Context, ep registry.Endpoint) record.Record {
	s.mu.Lock()
	s.started = append(s.started, ep.URL)
	s.mu.Unlock()
	select {
	case <-time.After(s.delay):
	case <-ctx.Done():
		return record.Record{URL: ep.URL, Outcome: record.OutcomeTimeout}
	}
	return record.Record{URL: ep.URL, Outcome: record.OutcomeReport}
}

func eps(urls ...string) []registry.Endpoint {
	out := make([]registry.Endpoint, 0, len(urls))
	for _, u := range urls {
		out = append(out, registry.Endpoint{URL: u})
	}
	return out
}

func TestRunChecksEveryEndpointOnce(t *testing.T) {
	s := &stub{}
	var progress strings.Builder
	r := &Runner{Checker: s, Workers: 3, Timeout: time.Second, Progress: &progress}
	var got []string
	err := r.Run(context.Background(), eps("https://a/1", "https://b/2", "https://c/3", "https://d/4"), func(rec record.Record) error {
		got = append(got, rec.URL)
		return nil
	})
	if err != nil || len(got) != 4 || len(s.started) != 4 {
		t.Fatalf("err %v, emitted %v, started %v", err, got, s.started)
	}
	if !strings.Contains(progress.String(), "[4/4] report") {
		t.Fatalf("progress:\n%s", progress.String())
	}
}

func TestRunSpacesChecksAgainstOneHost(t *testing.T) {
	s := &stub{}
	r := &Runner{Checker: s, Workers: 4, Timeout: time.Second, Limiter: limiter.New(60 * time.Millisecond)}
	start := time.Now()
	err := r.Run(context.Background(), eps("https://SAME/1", "https://same/2", "https://same/3"), func(record.Record) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	if el := time.Since(start); el < 110*time.Millisecond {
		t.Fatalf("three checks on one host took %v; the interval is 60ms", el)
	}
}

func TestRunAppliesTheDeadlinePerEndpoint(t *testing.T) {
	s := &stub{delay: time.Minute}
	r := &Runner{Checker: s, Workers: 2, Timeout: 20 * time.Millisecond}
	var outcomes []string
	err := r.Run(context.Background(), eps("https://a/1", "https://b/2"), func(rec record.Record) error {
		outcomes = append(outcomes, rec.Outcome)
		return nil
	})
	if err != nil || len(outcomes) != 2 || outcomes[0] != record.OutcomeTimeout {
		t.Fatalf("err %v outcomes %v", err, outcomes)
	}
}

func TestRunStopsWhenEmitFails(t *testing.T) {
	s := &stub{}
	r := &Runner{Checker: s, Workers: 1, Timeout: time.Second}
	boom := errors.New("disk full")
	err := r.Run(context.Background(), eps("https://a/1", "https://b/2", "https://c/3"), func(record.Record) error { return boom })
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v", err)
	}
}

func TestRunStopsWhenTheContextEnds(t *testing.T) {
	s := &stub{delay: time.Minute}
	r := &Runner{Checker: s, Workers: 1, Timeout: time.Hour, Limiter: limiter.New(time.Hour)}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	err := r.Run(ctx, eps("https://a/1", "https://a/2", "https://a/3"), func(record.Record) error { return nil })
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v", err)
	}
}

func TestHostOf(t *testing.T) {
	if got := hostOf("https://MCP.Example.com:8443/x"); got != "mcp.example.com" {
		t.Fatalf("hostOf = %q", got)
	}
	if got := hostOf("://bad"); got != "://bad" {
		t.Fatalf("hostOf(bad) = %q", got)
	}
}
