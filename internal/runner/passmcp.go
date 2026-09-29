// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// Package runner checks listed endpoints with the passmcp program: bounded
// concurrency, a per-host start interval, and a deadline per endpoint.
//
// It runs a released passmcp binary rather than linking passmcp's engine, so
// the census's method is exactly that release plus the flags in Args, and
// every safety property passmcp has holds here too.
package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"satellion.com/passmcp-census/internal/record"
	"satellion.com/passmcp-census/internal/registry"
)

// Phases are the only phases the census runs, the ones passmcp-registry
// runs: connectivity, discovery, handshake, protocol conformance and the
// catalogue. None of them invokes a tool. Execution, performance and
// resilience call tools or hold sessions open, and the auth phase presents
// a made-up credential; none of them runs against a server whose owner has
// not asked for it.
var Phases = []string{"net", "discovery", "handshake", "protocol", "catalog"}

// Passmcp runs one passmcp binary.
type Passmcp struct {
	Bin         string        // the passmcp binary
	UserAgent   string        // sent on every HTTP request passmcp makes
	RPS         float64       // passmcp's request throttle within a check
	CallTimeout time.Duration // passmcp's per-call timeout
}

// Args is the passmcp command line for an endpoint, exposed so the
// methodology and the tests state the same thing.
func (p *Passmcp) Args(endpoint string) []string {
	args := []string{"check", endpoint,
		"--phases", strings.Join(Phases, ","),
		"--auth", "none",
		"--output", "json",
		"--no-color",
		"--log-level", "error",
		"--rps", fmt.Sprint(p.RPS),
		"--timeout", p.CallTimeout.String(),
	}
	if p.UserAgent != "" {
		args = append(args, "--user-agent", p.UserAgent)
	}
	return args
}

// Version is what the binary reports as its version.
func (p *Passmcp) Version(ctx context.Context) (string, error) {
	out, stderr, code, err := p.run(ctx, "version")
	if err != nil {
		return "", err
	}
	if code != 0 {
		return "", fmt.Errorf("passmcp version exited %d: %s", code, record.Tail(string(stderr)))
	}
	v := strings.TrimSpace(string(out))
	return strings.TrimSpace(strings.TrimPrefix(v, "passmcp")), nil
}

// Check runs passmcp against one endpoint until it finishes or ctx ends.
func (p *Passmcp) Check(ctx context.Context, ep registry.Endpoint) record.Record {
	rec := record.Record{Name: ep.Name, Version: ep.Version, URL: ep.URL, Transport: ep.Transport}
	start := time.Now()
	out, stderr, code, err := p.run(ctx, p.Args(ep.URL)...)
	rec.DurationMS = time.Since(start).Milliseconds()
	rec.Exit = code
	switch {
	case ctx.Err() != nil:
		rec.Outcome, rec.Error = record.OutcomeTimeout, "the census's per-endpoint deadline ended the run"
	case err != nil:
		rec.Outcome, rec.Error = record.OutcomeError, record.Tail(err.Error())
	case code != 0 && code != 2:
		rec.Outcome, rec.Error = record.OutcomeError, fmt.Sprintf("passmcp exited %d: %s", code, record.Tail(string(stderr)))
	default:
		p.summarise(&rec, out)
	}
	return rec
}

func (p *Passmcp) summarise(rec *record.Record, report []byte) {
	s, err := record.Summarise(report)
	if err != nil {
		rec.Outcome, rec.Error = record.OutcomeError, record.Tail(err.Error())
		return
	}
	rec.Outcome, rec.Summary = record.OutcomeReport, s
}

func (p *Passmcp) run(ctx context.Context, args ...string) (stdout, stderr []byte, code int, err error) {
	cfg, cleanup, err := emptyConfig()
	if err != nil {
		return nil, nil, -1, err
	}
	defer cleanup()
	cmd := exec.CommandContext(ctx, p.Bin, args...) // #nosec G204 -- the operator names the binary; the arguments are ours
	cmd.Env = Environ(os.Environ(), cfg)
	cmd.WaitDelay = 5 * time.Second
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	runErr := cmd.Run()
	var exit *exec.ExitError
	switch {
	case runErr == nil:
		return out.Bytes(), errb.Bytes(), 0, nil
	case errors.As(runErr, &exit):
		return out.Bytes(), errb.Bytes(), exit.ExitCode(), nil
	default:
		return nil, errb.Bytes(), -1, fmt.Errorf("running passmcp: %w", runErr)
	}
}

// Environ is the environment passmcp runs with: the census's own, less
// every PASSMCP_ variable (a token or client secret the operator exported
// for their own use must not travel), plus PASSMCP_CONFIG naming an empty
// configuration file, so no profile can add a credential or switch on
// mutations.
func Environ(env []string, config string) []string {
	out := make([]string, 0, len(env)+2)
	for _, kv := range env {
		if !strings.HasPrefix(strings.ToUpper(kv), "PASSMCP_") {
			out = append(out, kv)
		}
	}
	return append(out, "PASSMCP_CONFIG="+config, "NO_COLOR=1")
}

func emptyConfig() (string, func(), error) {
	dir, err := os.MkdirTemp("", "passmcp-census-config-")
	if err != nil {
		return "", nil, err
	}
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte("{}\n"), 0o600); err != nil {
		_ = os.RemoveAll(dir)
		return "", nil, err
	}
	return path, func() { _ = os.RemoveAll(dir) }, nil
}
