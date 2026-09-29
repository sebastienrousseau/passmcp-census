// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// Package edition names the files of a census edition and carries the
// metadata a run records about itself: what was listed, what ran, with
// which passmcp, and how to run it again.
package edition

import (
	"fmt"
	"regexp"
	"time"
)

// Working files, written by run into the working directory. They name
// endpoints, so they are never published.
const (
	ListingFile = "listing.json"
	RecordsFile = "records.ndjson"
	RunFile     = "run.json"
)

var idPattern = regexp.MustCompile(`^[0-9]{4}-(0[1-9]|1[0-2])$`)

// Valid reports whether id is an edition id: a year and month, 2026-09.
func Valid(id string) error {
	if !idPattern.MatchString(id) {
		return fmt.Errorf("edition %q is not of the form YYYY-MM", id)
	}
	return nil
}

// Run is what one census run records about itself.
type Run struct {
	Edition       string `json:"edition"`
	CensusVersion string `json:"census_version"`
	// PassmcpVersion is what the binary said when asked, before the run.
	PassmcpVersion string `json:"passmcp_version"`
	Registry       string `json:"registry"`
	// The registry snapshot: when the listing started and finished.
	SnapshotStarted  time.Time `json:"snapshot_started"`
	SnapshotFinished time.Time `json:"snapshot_finished"`
	RunStarted       time.Time `json:"run_started"`
	RunFinished      time.Time `json:"run_finished"`
	DurationSeconds  int64     `json:"duration_seconds"`
	// Complete is false when the run was interrupted or limited, and the
	// dataset then covers only the endpoints that were checked.
	Complete  bool   `json:"complete"`
	Interrupt string `json:"interrupt,omitempty"`

	Parameters Parameters `json:"parameters"`
	Listing    Listing    `json:"listing"`
	// PassmcpArgs is the command passmcp ran for each endpoint.
	PassmcpArgs []string `json:"passmcp_args"`
	// Reproduce is the command that runs this edition again.
	Reproduce string `json:"reproduce"`
}

// Parameters are the run's politeness and scope settings.
type Parameters struct {
	Workers         int     `json:"workers"`
	HostInterval    string  `json:"host_interval"`
	EndpointTimeout string  `json:"endpoint_timeout"`
	RPS             float64 `json:"rps"`
	CallTimeout     string  `json:"call_timeout"`
	UserAgent       string  `json:"user_agent"`
	Limit           int     `json:"limit"`
	Exclusions      int     `json:"exclusions"`
}

// Listing counts what the registry snapshot held and what became of it.
type Listing struct {
	Entries    int            `json:"entries"`
	Current    int            `json:"current"`
	Endpoints  int            `json:"endpoints"`
	Skipped    map[string]int `json:"skipped"`
	Duplicates int            `json:"duplicates"`
	Excluded   int            `json:"excluded"`
	Selected   int            `json:"selected"`
	Checked    int            `json:"checked"`
}
