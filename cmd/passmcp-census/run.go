// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"satellion.com/passmcp-census/internal/edition"
	"satellion.com/passmcp-census/internal/exclude"
	"satellion.com/passmcp-census/internal/limiter"
	"satellion.com/passmcp-census/internal/record"
	"satellion.com/passmcp-census/internal/registry"
	"satellion.com/passmcp-census/internal/runner"
)

// DefaultRegistry is the official MCP Registry.
const DefaultRegistry = "https://registry.modelcontextprotocol.io"

// now is the clock; tests replace it.
var now = time.Now

// registryBackoff is the first wait before asking the registry again for
// a page that failed; tests shorten it.
var registryBackoff = 5 * time.Second

type runOptions struct {
	edition, registry, passmcp, work, exclusions string
	workers, limit                               int
	hostInterval, timeout, callTimeout           time.Duration
	rps                                          float64
}

func runFlags(stderr io.Writer) (*flag.FlagSet, *runOptions) {
	o := &runOptions{}
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&o.edition, "edition", "", "edition id, YYYY-MM (required)")
	fs.StringVar(&o.registry, "registry", DefaultRegistry, "MCP Registry base URL")
	fs.StringVar(&o.passmcp, "passmcp", "passmcp", "passmcp binary to run")
	fs.StringVar(&o.work, "work", "", "private working directory (default build/census/<edition>)")
	fs.StringVar(&o.exclusions, "exclusions", "data/exclusions.txt", "servers whose owners asked not to be contacted")
	fs.IntVar(&o.workers, "workers", 4, "endpoints checked at once")
	fs.IntVar(&o.limit, "limit", 0, "check at most this many endpoints (0: all); a limited run is marked incomplete")
	fs.DurationVar(&o.hostInterval, "host-interval", 10*time.Second, "minimum time between checks starting against one host")
	fs.DurationVar(&o.timeout, "timeout", 3*time.Minute, "deadline for one endpoint's check")
	fs.DurationVar(&o.callTimeout, "call-timeout", 20*time.Second, "passmcp's per-call timeout")
	fs.Float64Var(&o.rps, "rps", 1, "passmcp's maximum requests per second within a check")
	return fs, o
}

func (o *runOptions) validate() error {
	if err := edition.Valid(o.edition); err != nil {
		return err
	}
	if o.workers < 1 || o.limit < 0 || o.timeout <= 0 || o.callTimeout <= 0 || o.hostInterval < 0 {
		return errors.New("workers must be at least 1, and limit, the interval and timeouts not negative")
	}
	if o.rps <= 0 || o.rps > 2 {
		return errors.New("rps must be above 0 and at most 2: the census is polite by construction")
	}
	if o.work == "" {
		o.work = filepath.Join("build", "census", o.edition)
	}
	return nil
}

func cmdRun(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs, o := runFlags(stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if err := o.validate(); err != nil {
		_, _ = fmt.Fprintln(stderr, "passmcp-census run:", err)
		return 2
	}
	meta, err := census(ctx, o, stderr)
	if err != nil {
		return fail(stderr, err)
	}
	b, _ := json.MarshalIndent(meta, "", "  ")
	_, _ = fmt.Fprintln(stdout, string(b))
	if !meta.Complete {
		return 1
	}
	return 0
}

// cmdList is a run's first step alone: the registry snapshot and the
// selection, with no server contacted.
func cmdList(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs, o := runFlags(stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if err := o.validate(); err != nil {
		_, _ = fmt.Fprintln(stderr, "passmcp-census list:", err)
		return 2
	}
	excl, err := loadExclusions(o.exclusions)
	if err != nil {
		return fail(stderr, err)
	}
	if err := os.MkdirAll(o.work, 0o750); err != nil {
		return fail(stderr, err)
	}
	meta := newRun(o, excl.Len())
	if _, err := snapshot(ctx, o, excl, &meta, stderr); err != nil {
		return fail(stderr, err)
	}
	b, _ := json.MarshalIndent(meta.Listing, "", "  ")
	_, _ = fmt.Fprintln(stdout, string(b))
	return 0
}

// census is one run: the passmcp version, the registry snapshot, the
// checks, and the run's own record, all in the working directory.
func census(ctx context.Context, o *runOptions, stderr io.Writer) (edition.Run, error) {
	excl, err := loadExclusions(o.exclusions)
	if err != nil {
		return edition.Run{}, err
	}
	pm := &runner.Passmcp{Bin: o.passmcp, UserAgent: userAgent(), RPS: o.rps, CallTimeout: o.callTimeout}
	meta := newRun(o, excl.Len())
	if meta.PassmcpVersion, err = pm.Version(ctx); err != nil {
		return meta, err
	}
	if err := os.MkdirAll(o.work, 0o750); err != nil {
		return meta, err
	}
	eps, err := snapshot(ctx, o, excl, &meta, stderr)
	if err != nil {
		return meta, err
	}
	meta.PassmcpArgs = pm.Args("<endpoint>")
	_, _ = fmt.Fprintf(stderr, "passmcp-census: checking %d endpoints with passmcp %s\n", len(eps), meta.PassmcpVersion)
	runErr := check(ctx, o, pm, eps, &meta, stderr)
	meta.RunFinished = now().UTC()
	meta.DurationSeconds = int64(meta.RunFinished.Sub(meta.RunStarted).Seconds())
	meta.Complete = runErr == nil && o.limit == 0 && meta.Listing.Checked == meta.Listing.Selected
	if runErr != nil {
		meta.Interrupt = runErr.Error()
	}
	return meta, writeJSON(filepath.Join(o.work, edition.RunFile), meta)
}

func newRun(o *runOptions, exclusions int) edition.Run {
	return edition.Run{
		Edition:       o.edition,
		CensusVersion: Version,
		Registry:      o.registry,
		RunStarted:    now().UTC(),
		Parameters: edition.Parameters{
			Workers: o.workers, HostInterval: o.hostInterval.String(), EndpointTimeout: o.timeout.String(),
			RPS: o.rps, CallTimeout: o.callTimeout.String(), UserAgent: userAgent(), Limit: o.limit,
			Exclusions: exclusions,
		},
		Reproduce: reproduce(o),
	}
}

func userAgent() string {
	return "passmcp-census/" + Version + " (+https://github.com/sebastienrousseau/passmcp-census)"
}

// reproduce is the command that runs this edition again, from a checkout
// of this repository with passmcp on PATH.
func reproduce(o *runOptions) string {
	parts := []string{"make census", "EDITION=" + o.edition}
	if o.registry != DefaultRegistry {
		parts = append(parts, "REGISTRY="+o.registry)
	}
	return strings.Join(parts, " ")
}

// snapshot lists the registry and selects the endpoints to check.
func snapshot(ctx context.Context, o *runOptions, excl *exclude.List, meta *edition.Run, stderr io.Writer) ([]registry.Endpoint, error) {
	meta.SnapshotStarted = now().UTC()
	c := &registry.Client{Base: o.registry, UserAgent: userAgent(), Retries: 4, Backoff: registryBackoff, Progress: stderr}
	l, err := c.List(ctx)
	meta.SnapshotFinished = now().UTC()
	if err != nil {
		return nil, err
	}
	if err := writeJSON(filepath.Join(o.work, edition.ListingFile), l); err != nil {
		return nil, err
	}
	return selectEndpoints(l, excl, o.limit, &meta.Listing), nil
}

// selectEndpoints drops duplicate URLs and excluded servers, and applies
// the limit, counting each step.
func selectEndpoints(l registry.Listing, excl *exclude.List, limit int, counts *edition.Listing) []registry.Endpoint {
	counts.Entries, counts.Current, counts.Endpoints = l.Entries, l.Current, len(l.Endpoints)
	counts.Skipped = map[string]int{}
	for _, s := range l.Skipped {
		counts.Skipped[s.Reason]++
	}
	eps, dup := registry.Distinct(l.Endpoints)
	counts.Duplicates = dup
	kept := eps[:0]
	for _, e := range eps {
		if excl.Excludes(e.Name) {
			counts.Excluded++
			continue
		}
		kept = append(kept, e)
	}
	if limit > 0 && len(kept) > limit {
		kept = kept[:limit]
	}
	counts.Selected = len(kept)
	return kept
}

func check(ctx context.Context, o *runOptions, pm runner.Checker, eps []registry.Endpoint, meta *edition.Run, stderr io.Writer) error {
	f, err := os.Create(filepath.Join(o.work, edition.RecordsFile)) // #nosec G304 -- the operator's own working directory
	if err != nil {
		return err
	}
	w := bufio.NewWriter(f)
	enc := json.NewEncoder(w)
	r := &runner.Runner{Checker: pm, Limiter: limiter.New(o.hostInterval), Workers: o.workers, Timeout: o.timeout, Progress: stderr}
	runErr := r.Run(ctx, eps, func(rec record.Record) error {
		meta.Listing.Checked++
		return enc.Encode(rec)
	})
	return errors.Join(runErr, w.Flush(), f.Close())
}

func loadExclusions(path string) (*exclude.List, error) {
	f, err := os.Open(path) // #nosec G304 -- the operator's own flag
	if err != nil {
		return nil, fmt.Errorf("exclusions: %w", err)
	}
	defer func() { _ = f.Close() }()
	return exclude.Parse(f)
}

func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o600)
}
