// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package aggregate

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
)

// The published files of an edition.
const (
	CensusFile      = "census.json"
	OutcomesFile    = "outcomes.csv"
	ListingFile     = "listing.csv"
	PhasesFile      = "by-phase.csv"
	ChecksFile      = "by-check.csv"
	TransportFile   = "by-transport.csv"
	AuthFile        = "by-auth.csv"
	DataPackageFile = "datapackage.json"
)

// table is one CSV file: its header and rows, every value a string.
type table struct {
	name   string
	header []string
	rows   [][]string
}

var statusHeader = []string{"endpoints", "pass", "warn", "fail", "info", "skip", "fail_rate"}
var groupHeader = []string{"endpoints", "reports", "errors", "timeouts", "blocked", "with_failure", "failure_rate"}

func (d Dataset) tables() []table {
	return []table{
		{OutcomesFile, []string{"outcome", "endpoints", "share"}, outcomeRows(d.Outcomes)},
		{ListingFile, []string{"stage", "reason", "count"}, listingRows(d.Listing)},
		{PhasesFile, append([]string{"phase"}, statusHeader...), statusCells(d.Phases, false)},
		{ChecksFile, append([]string{"check_id", "phase"}, statusHeader...), statusCells(d.Checks, true)},
		{TransportFile, append([]string{"transport"}, groupHeader...), groupCells(d.Transport)},
		{AuthFile, append([]string{"auth_scheme"}, groupHeader...), groupCells(d.Auth)},
	}
}

// Write writes the edition's files into dir and returns their names.
func (d Dataset) Write(dir string) ([]string, error) {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, err
	}
	var names []string
	for _, t := range d.tables() {
		if err := writeFile(dir, t.name, t.csv()); err != nil {
			return nil, err
		}
		names = append(names, t.name)
	}
	for name, v := range map[string]any{CensusFile: d, DataPackageFile: d.descriptor()} {
		b, err := marshal(v)
		if err != nil {
			return nil, err
		}
		if err := writeFile(dir, name, b); err != nil {
			return nil, err
		}
	}
	return append(names, CensusFile, DataPackageFile), nil
}

func writeFile(dir, name string, b []byte) error {
	return os.WriteFile(filepath.Join(dir, name), b, 0o600)
}

func marshal(v any) ([]byte, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

func (t table) csv() []byte {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write(t.header)
	_ = w.WriteAll(t.rows) // WriteAll flushes; a bytes.Buffer cannot fail
	return buf.Bytes()
}

func outcomeRows(rows []OutcomeRow) [][]string {
	out := make([][]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, []string{r.Outcome, itoa(r.Endpoints), rate(r.Share)})
	}
	return out
}

func listingRows(rows []ListingRow) [][]string {
	out := make([][]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, []string{r.Stage, r.Reason, itoa(r.Count)})
	}
	return out
}

func statusCells(rows []StatusRow, withPhase bool) [][]string {
	out := make([][]string, 0, len(rows))
	for _, r := range rows {
		lead := []string{r.ID}
		if withPhase {
			lead = append(lead, r.Phase)
		}
		out = append(out, append(lead, itoa(r.Endpoints), itoa(r.Pass), itoa(r.Warn),
			itoa(r.Fail), itoa(r.Info), itoa(r.Skip), rate(r.FailRate)))
	}
	return out
}

func groupCells(rows []GroupRow) [][]string {
	out := make([][]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, []string{r.Group, itoa(r.Endpoints), itoa(r.Reports), itoa(r.Errors),
			itoa(r.Timeouts), itoa(r.Blocked), itoa(r.WithFailure), rate(r.FailureRate)})
	}
	return out
}

func itoa(n int) string { return strconv.Itoa(n) }

// rate renders a rate, or an empty cell when there is none.
func rate(r *float64) string {
	if r == nil {
		return ""
	}
	return strconv.FormatFloat(*r, 'f', 4, 64)
}
