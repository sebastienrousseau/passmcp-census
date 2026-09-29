// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package limiter

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestWaitSpacesOneHostAndNotOthers(t *testing.T) {
	l := New(50 * time.Millisecond)
	ctx := context.Background()
	start := time.Now()
	for _, h := range []string{"a", "b", "c"} {
		if err := l.Wait(ctx, h); err != nil {
			t.Fatal(err)
		}
	}
	if el := time.Since(start); el > 40*time.Millisecond {
		t.Fatalf("three hosts waited %v; different hosts are not spaced", el)
	}
	if err := l.Wait(ctx, "a"); err != nil {
		t.Fatal(err)
	}
	if el := time.Since(start); el < 45*time.Millisecond {
		t.Fatalf("the second check on a started after %v, before the interval", el)
	}
}

func TestWaitEndsWithTheContext(t *testing.T) {
	l := New(time.Hour)
	_ = l.Wait(context.Background(), "a")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if err := l.Wait(ctx, "a"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v", err)
	}
	done, cancel2 := context.WithCancel(context.Background())
	cancel2()
	if err := l.Wait(done, "fresh"); !errors.Is(err, context.Canceled) {
		t.Fatalf("an ended context must be reported even with no wait: %v", err)
	}
}

func TestWaitUsesItsClock(t *testing.T) {
	l := New(time.Second)
	fixed := time.Unix(1000, 0)
	l.now = func() time.Time { return fixed }
	_ = l.Wait(context.Background(), "a")
	if got := l.next["a"]; !got.Equal(fixed.Add(time.Second)) {
		t.Fatalf("next = %v", got)
	}
}
