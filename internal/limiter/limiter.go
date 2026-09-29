// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// Package limiter spaces out the checks the census starts against one host.
//
// passmcp throttles the requests inside a check (--rps). This is the other
// half: a provider serving many listed endpoints from one hostname sees the
// checks against them start no closer together than the interval, however
// many workers the census runs.
package limiter

import (
	"context"
	"sync"
	"time"
)

// PerHost hands out start times, one host at a time.
type PerHost struct {
	interval time.Duration
	now      func() time.Time

	mu   sync.Mutex
	next map[string]time.Time
}

// New returns a limiter that starts at most one check per host per
// interval.
func New(interval time.Duration) *PerHost {
	return &PerHost{interval: interval, now: time.Now, next: map[string]time.Time{}}
}

// Wait blocks until a check against host may start, or ctx ends.
func (l *PerHost) Wait(ctx context.Context, host string) error {
	l.mu.Lock()
	now := l.now()
	at := l.next[host]
	if at.Before(now) {
		at = now
	}
	l.next[host] = at.Add(l.interval)
	l.mu.Unlock()

	d := at.Sub(now)
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
