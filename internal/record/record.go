// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// Package record is what the census keeps about one checked endpoint: the
// outcome of the passmcp run, and passmcp's JSON report reduced to one status
// per phase and one per check id.
//
// A Record names the endpoint it is about, so records are working data and
// stay in the run's directory. Only the aggregate package's tables are
// published.
package record

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Outcomes of one check.
const (
	OutcomeReport  = "report"  // passmcp produced a report (exit 0 or 2)
	OutcomeError   = "error"   // passmcp ran and produced no report
	OutcomeTimeout = "timeout" // the census's per-endpoint deadline ended it
)

// Authorization schemes, as observed without credentials.
const (
	AuthNone      = "none"      // the server answered without asking for credentials
	AuthOAuth     = "oauth"     // it asked, and advertised OAuth metadata
	AuthOther     = "other"     // it asked, and advertised no OAuth metadata
	AuthUnreached = "unreached" // passmcp never reached it, so it is not known
)

// Unrecognised stands in for a check id or phase name that is not of the
// form passmcp gives them, so server-chosen text never reaches a table.
const Unrecognised = "unrecognised"

// SchemaVersion is the passmcp JSON report format this package reads.
const SchemaVersion = 1

// maxError bounds the passmcp error text a record keeps.
const maxError = 400

// Record is one endpoint's result.
type Record struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	URL       string `json:"url"`
	Transport string `json:"transport"`

	Outcome    string `json:"outcome"`
	Exit       int    `json:"exit"`
	Error      string `json:"error,omitempty"`
	DurationMS int64  `json:"duration_ms"`

	// Summary is filled when Outcome is OutcomeReport.
	*Summary
}

// Summary is a passmcp report reduced to what the census counts.
type Summary struct {
	PassmcpVersion string            `json:"passmcp_version"`
	AuthScheme     string            `json:"auth_scheme"`
	Blocked        bool              `json:"blocked"`
	Phases         map[string]string `json:"phases"`
	Checks         map[string]Check  `json:"checks"`
}

// Check is one check id's status on one endpoint.
type Check struct {
	Phase  string `json:"phase"`
	Status string `json:"status"`
}

type report struct {
	Passmcp struct {
		Version       string `json:"version"`
		SchemaVersion int    `json:"schema_version"`
	} `json:"passmcp"`
	Auth struct {
		Reached   bool   `json:"reached"`
		Required  bool   `json:"required"`
		Issuer    string `json:"issuer"`
		PRMSource string `json:"prm_source"`
	} `json:"auth"`
	Blocked string `json:"blocked"`
	Phases  []struct {
		Name     string `json:"name"`
		Status   string `json:"status"`
		Findings []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"findings"`
	} `json:"phases"`
}

// rank orders statuses from least to most telling: a check id that appears
// more than once on one endpoint (one finding per tool, say) counts once,
// at its worst.
var rank = map[string]int{"skip": 1, "info": 2, "pass": 3, "warn": 4, "fail": 5}

// Statuses are the finding statuses, in the order tables list them.
var Statuses = []string{"pass", "warn", "fail", "info", "skip"}

var idPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_.-]{0,79}$`)

// Summarise reduces a passmcp JSON report.
func Summarise(b []byte) (*Summary, error) {
	var r report
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, fmt.Errorf("not a passmcp JSON report: %w", err)
	}
	if r.Passmcp.SchemaVersion != SchemaVersion {
		return nil, fmt.Errorf("passmcp report schema %d; this census reads %d", r.Passmcp.SchemaVersion, SchemaVersion)
	}
	if len(r.Phases) == 0 {
		return nil, errors.New("the passmcp report has no phases")
	}
	s := &Summary{
		PassmcpVersion: Clean(r.Passmcp.Version),
		AuthScheme:     authScheme(r.Auth.Reached, r.Auth.Required, r.Auth.Issuer != "" || r.Auth.PRMSource != ""),
		Blocked:        r.Blocked != "",
		Phases:         map[string]string{},
		Checks:         map[string]Check{},
	}
	for _, p := range r.Phases {
		phase := Clean(p.Name)
		if _, ok := rank[p.Status]; ok {
			s.Phases[phase] = p.Status
		}
		for _, f := range p.Findings {
			s.add(Clean(f.ID), phase, f.Status)
		}
	}
	return s, nil
}

func (s *Summary) add(id, phase, status string) {
	if _, ok := rank[status]; !ok {
		return
	}
	if prev, ok := s.Checks[id]; ok && rank[prev.Status] >= rank[status] {
		return
	}
	s.Checks[id] = Check{Phase: phase, Status: status}
}

func authScheme(reached, required, oauth bool) string {
	switch {
	case !reached:
		return AuthUnreached
	case !required:
		return AuthNone
	case oauth:
		return AuthOAuth
	default:
		return AuthOther
	}
}

// Clean returns a check id or phase name in the form passmcp gives them, or
// Unrecognised. A computed family id (auth.source.<field>) is reduced to its
// family, because the value after it is the server's.
func Clean(id string) string {
	if strings.HasPrefix(id, "auth.source.") {
		return "auth.source"
	}
	if !idPattern.MatchString(id) {
		return Unrecognised
	}
	return id
}

// Tail bounds error text to its last maxError bytes, on a rune boundary.
func Tail(s string) string {
	s = strings.TrimSpace(s)
	if len(s) <= maxError {
		return s
	}
	cut := len(s) - maxError
	for cut < len(s) && (s[cut]&0xC0) == 0x80 {
		cut++
	}
	return "…" + s[cut:]
}
