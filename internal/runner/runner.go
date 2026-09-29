// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package runner

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"sync"
	"time"

	"satellion.com/passmcp-census/internal/limiter"
	"satellion.com/passmcp-census/internal/record"
	"satellion.com/passmcp-census/internal/registry"
)

// Checker checks one endpoint. Passmcp is the real one.
type Checker interface {
	Check(ctx context.Context, ep registry.Endpoint) record.Record
}

// Runner checks a list of endpoints.
type Runner struct {
	Checker  Checker
	Limiter  *limiter.PerHost
	Workers  int           // checks in flight at once
	Timeout  time.Duration // deadline for one endpoint, from its start
	Progress io.Writer     // one line per finished endpoint; nil for none
}

// Run checks every endpoint and hands each record to emit, one at a time,
// in the order they finish. It stops early only when ctx ends or emit
// fails; an endpoint that fails to check is a record, not an error.
func (r *Runner) Run(ctx context.Context, eps []registry.Endpoint, emit func(record.Record) error) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	jobs := make(chan registry.Endpoint)
	results := make(chan record.Record)
	var wg sync.WaitGroup
	for range max(r.Workers, 1) {
		wg.Go(func() { r.work(ctx, jobs, results) })
	}
	go func() {
		defer close(jobs)
		for _, ep := range eps {
			select {
			case jobs <- ep:
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() { wg.Wait(); close(results) }()
	return r.collect(ctx, cancel, results, len(eps), emit)
}

func (r *Runner) work(ctx context.Context, jobs <-chan registry.Endpoint, results chan<- record.Record) {
	for ep := range jobs {
		if r.Limiter != nil && r.Limiter.Wait(ctx, hostOf(ep.URL)) != nil {
			return
		}
		c, cancel := context.WithTimeout(ctx, r.Timeout)
		rec := r.Checker.Check(c, ep)
		cancel()
		select {
		case results <- rec:
		case <-ctx.Done():
			return
		}
	}
}

func (r *Runner) collect(ctx context.Context, cancel func(), results <-chan record.Record, total int, emit func(record.Record) error) error {
	var err error
	done := 0
	for rec := range results {
		if err != nil {
			continue // drain, so every worker can exit
		}
		done++
		if err = emit(rec); err != nil {
			cancel()
			continue
		}
		if r.Progress != nil {
			_, _ = fmt.Fprintf(r.Progress, "[%d/%d] %s %s (%d ms)\n", done, total, rec.Outcome, hostOf(rec.URL), rec.DurationMS)
		}
	}
	if err != nil {
		return err
	}
	return ctx.Err()
}

// hostOf is the host a URL names, lower-cased; the limiter's key.
func hostOf(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	return strings.ToLower(u.Hostname())
}
