// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// Package aggregate turns a run's per-endpoint records into the published
// dataset: counts and rates by outcome, check, phase, transport and
// authorization scheme. Every table is one-dimensional and carries no
// endpoint's name, URL or host, so nothing published can point at a
// server.
package aggregate

import (
	"sort"
	"strconv"

	"satellion.com/passmcp-census/internal/edition"
	"satellion.com/passmcp-census/internal/record"
)

// Dataset is one edition's published data.
type Dataset struct {
	Edition   string        `json:"edition"`
	Run       edition.Run   `json:"run"`
	Outcomes  []OutcomeRow  `json:"outcomes"`
	Listing   []ListingRow  `json:"listing"`
	Phases    []StatusRow   `json:"phases"`
	Checks    []StatusRow   `json:"checks"`
	Transport []GroupRow    `json:"transports"`
	Auth      []GroupRow    `json:"auth_schemes"`
	Totals    OutcomeTotals `json:"totals"`
}

// OutcomeTotals are the headline counts.
type OutcomeTotals struct {
	Checked     int      `json:"checked"`
	Reports     int      `json:"reports"`
	WithFailure int      `json:"with_failure"`
	FailureRate *float64 `json:"failure_rate"`
}

// OutcomeRow counts endpoints by what their check produced.
type OutcomeRow struct {
	Outcome   string   `json:"outcome"`
	Endpoints int      `json:"endpoints"`
	Share     *float64 `json:"share"`
}

// ListingRow counts what the registry snapshot held at one stage.
type ListingRow struct {
	Stage  string `json:"stage"`
	Reason string `json:"reason"`
	Count  int    `json:"count"`
}

// StatusRow counts endpoints by status for one phase or one check.
type StatusRow struct {
	ID        string   `json:"id"`
	Phase     string   `json:"phase"`
	Endpoints int      `json:"endpoints"`
	Pass      int      `json:"pass"`
	Warn      int      `json:"warn"`
	Fail      int      `json:"fail"`
	Info      int      `json:"info"`
	Skip      int      `json:"skip"`
	FailRate  *float64 `json:"fail_rate"`
}

// GroupRow counts endpoints in one transport or authorization scheme.
type GroupRow struct {
	Group       string   `json:"group"`
	Endpoints   int      `json:"endpoints"`
	Reports     int      `json:"reports"`
	Errors      int      `json:"errors"`
	Timeouts    int      `json:"timeouts"`
	Blocked     int      `json:"blocked"`
	WithFailure int      `json:"with_failure"`
	FailureRate *float64 `json:"failure_rate"`
}

// Build aggregates the records of one run.
func Build(run edition.Run, recs []record.Record) Dataset {
	d := Dataset{Edition: run.Edition, Run: run}
	d.Outcomes = outcomes(recs)
	d.Listing = listing(run.Listing)
	d.Phases = statusRows(recs, phaseStatuses)
	d.Checks = statusRows(recs, checkStatuses)
	d.Transport = groups(recs, func(r record.Record) string { return r.Transport })
	d.Auth = groups(reportsOnly(recs), func(r record.Record) string { return r.AuthScheme })
	d.Totals = totals(recs)
	return d
}

func totals(recs []record.Record) OutcomeTotals {
	t := OutcomeTotals{Checked: len(recs)}
	for _, r := range recs {
		if r.Summary == nil {
			continue
		}
		t.Reports++
		if hasFailure(r.Summary) {
			t.WithFailure++
		}
	}
	t.FailureRate = Rate(t.WithFailure, t.Reports)
	return t
}

func outcomes(recs []record.Record) []OutcomeRow {
	counts := map[string]int{}
	for _, r := range recs {
		counts[r.Outcome]++
	}
	rows := make([]OutcomeRow, 0, 3)
	for _, o := range []string{record.OutcomeReport, record.OutcomeError, record.OutcomeTimeout} {
		rows = append(rows, OutcomeRow{Outcome: o, Endpoints: counts[o], Share: Rate(counts[o], len(recs))})
	}
	return rows
}

func listing(l edition.Listing) []ListingRow {
	rows := []ListingRow{
		{Stage: "entries", Count: l.Entries},
		{Stage: "current", Count: l.Current},
		{Stage: "endpoints", Count: l.Endpoints},
	}
	reasons := make([]string, 0, len(l.Skipped))
	for r := range l.Skipped {
		reasons = append(reasons, r)
	}
	sort.Strings(reasons)
	for _, r := range reasons {
		rows = append(rows, ListingRow{Stage: "skipped", Reason: r, Count: l.Skipped[r]})
	}
	return append(rows,
		ListingRow{Stage: "duplicates", Count: l.Duplicates},
		ListingRow{Stage: "excluded", Count: l.Excluded},
		ListingRow{Stage: "selected", Count: l.Selected},
		ListingRow{Stage: "checked", Count: l.Checked},
	)
}

// statusOf yields, for one record, each id with its phase and status.
type statusOf func(s *record.Summary, yield func(id, phase, status string))

func phaseStatuses(s *record.Summary, yield func(id, phase, status string)) {
	for p, st := range s.Phases {
		yield(p, p, st)
	}
}

func checkStatuses(s *record.Summary, yield func(id, phase, status string)) {
	for id, c := range s.Checks {
		yield(id, c.Phase, c.Status)
	}
}

func statusRows(recs []record.Record, of statusOf) []StatusRow {
	rows := map[string]*StatusRow{}
	for _, r := range recs {
		if r.Summary == nil {
			continue
		}
		of(r.Summary, func(id, phase, status string) {
			row := rows[id]
			if row == nil {
				row = &StatusRow{ID: id, Phase: phase}
				rows[id] = row
			}
			row.count(status)
		})
	}
	out := make([]StatusRow, 0, len(rows))
	for _, row := range rows {
		row.FailRate = Rate(row.Fail, row.Pass+row.Warn+row.Fail)
		out = append(out, *row)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Phase != out[j].Phase {
			return out[i].Phase < out[j].Phase
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func (row *StatusRow) count(status string) {
	row.Endpoints++
	switch status {
	case "pass":
		row.Pass++
	case "warn":
		row.Warn++
	case "fail":
		row.Fail++
	case "info":
		row.Info++
	default:
		row.Skip++
	}
}

func groups(recs []record.Record, key func(record.Record) string) []GroupRow {
	rows := map[string]*GroupRow{}
	for _, r := range recs {
		k := key(r)
		row := rows[k]
		if row == nil {
			row = &GroupRow{Group: k}
			rows[k] = row
		}
		row.add(r)
	}
	out := make([]GroupRow, 0, len(rows))
	for _, row := range rows {
		row.FailureRate = Rate(row.WithFailure, row.Reports)
		out = append(out, *row)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Group < out[j].Group })
	return out
}

func (row *GroupRow) add(r record.Record) {
	row.Endpoints++
	switch r.Outcome {
	case record.OutcomeError:
		row.Errors++
	case record.OutcomeTimeout:
		row.Timeouts++
	}
	if r.Summary == nil {
		return
	}
	row.Reports++
	if r.Blocked {
		row.Blocked++
	}
	if hasFailure(r.Summary) {
		row.WithFailure++
	}
}

func reportsOnly(recs []record.Record) []record.Record {
	out := make([]record.Record, 0, len(recs))
	for _, r := range recs {
		if r.Summary != nil {
			out = append(out, r)
		}
	}
	return out
}

func hasFailure(s *record.Summary) bool {
	for _, c := range s.Checks {
		if c.Status == "fail" {
			return true
		}
	}
	return false
}

// Rate is n over of, rounded to four decimal places, or nil when of is 0:
// a rate over nothing is not zero, it is absent.
func Rate(n, of int) *float64 {
	if of == 0 {
		return nil
	}
	v, _ := strconv.ParseFloat(strconv.FormatFloat(float64(n)/float64(of), 'f', 4, 64), 64)
	return &v
}
