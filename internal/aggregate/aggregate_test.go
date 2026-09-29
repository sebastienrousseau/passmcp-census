// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package aggregate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"satellion.com/passmcp-census/internal/edition"
	"satellion.com/passmcp-census/internal/record"
)

func summary(auth string, blocked bool, phases map[string]string, checks map[string]record.Check) *record.Summary {
	return &record.Summary{PassmcpVersion: "0.0.1", AuthScheme: auth, Blocked: blocked, Phases: phases, Checks: checks}
}

// fixture is four endpoints: two reports (one with a failure), one error
// and one timeout, over two transports.
func fixture() (edition.Run, []record.Record) {
	run := edition.Run{
		Edition: "2026-09", CensusVersion: "0.0.2", PassmcpVersion: "0.0.1",
		Registry:    "https://registry.example",
		RunFinished: time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC),
		Listing: edition.Listing{
			Entries: 10, Current: 6, Endpoints: 5, Duplicates: 1, Excluded: 0, Selected: 4, Checked: 4,
			Skipped: map[string]int{"templated URL": 2, "requires headers the user supplies": 1},
		},
	}
	recs := []record.Record{
		{Name: "io.secret/one", URL: "https://one.secret.example/mcp", Transport: "streamable-http", Outcome: record.OutcomeReport,
			Summary: summary(record.AuthNone, false,
				map[string]string{"net": "pass", "protocol": "fail"},
				map[string]record.Check{"net.dns": {Phase: "net", Status: "pass"}, "protocol.origin": {Phase: "protocol", Status: "fail"}})},
		{Name: "io.secret/two", URL: "https://two.secret.example/mcp", Transport: "streamable-http", Outcome: record.OutcomeReport,
			Summary: summary(record.AuthOAuth, true,
				map[string]string{"net": "pass", "protocol": "skip"},
				map[string]record.Check{"net.dns": {Phase: "net", Status: "pass"}, "protocol.origin": {Phase: "protocol", Status: "skip"}})},
		{Name: "io.secret/three", URL: "https://three.secret.example/sse", Transport: "sse", Outcome: record.OutcomeError, Error: "no such host three.secret.example"},
		{Name: "io.secret/four", URL: "https://four.secret.example/mcp", Transport: "streamable-http", Outcome: record.OutcomeTimeout},
	}
	return run, recs
}

func TestBuild(t *testing.T) {
	d := Build(fixture())
	if d.Totals.Checked != 4 || d.Totals.Reports != 2 || d.Totals.WithFailure != 1 || *d.Totals.FailureRate != 0.5 {
		t.Fatalf("totals = %+v", d.Totals)
	}
	if o := d.Outcomes; len(o) != 3 || o[0].Endpoints != 2 || *o[0].Share != 0.5 || o[1].Endpoints != 1 || o[2].Endpoints != 1 {
		t.Fatalf("outcomes = %+v", o)
	}
	origin := find(t, d.Checks, "protocol.origin")
	if origin.Endpoints != 2 || origin.Fail != 1 || origin.Skip != 1 || *origin.FailRate != 1 {
		t.Fatalf("protocol.origin = %+v", origin)
	}
	if dns := find(t, d.Checks, "net.dns"); dns.Pass != 2 || *dns.FailRate != 0 {
		t.Fatalf("net.dns = %+v", dns)
	}
	if len(d.Phases) != 2 || d.Phases[0].ID != "net" {
		t.Fatalf("phases = %+v", d.Phases)
	}
	if len(d.Transport) != 2 || d.Transport[0].Group != "sse" || d.Transport[0].Errors != 1 ||
		d.Transport[1].Timeouts != 1 || d.Transport[1].Reports != 2 || d.Transport[1].Blocked != 1 {
		t.Fatalf("transports = %+v", d.Transport)
	}
	if len(d.Auth) != 2 || d.Auth[0].Group != record.AuthNone || d.Auth[0].WithFailure != 1 || d.Auth[1].Blocked != 1 {
		t.Fatalf("auth = %+v", d.Auth)
	}
}

func find(t *testing.T, rows []StatusRow, id string) StatusRow {
	t.Helper()
	for _, r := range rows {
		if r.ID == id {
			return r
		}
	}
	t.Fatalf("no row %q", id)
	return StatusRow{}
}

func TestRateOverNothingIsAbsent(t *testing.T) {
	if Rate(0, 0) != nil {
		t.Fatal("a rate over nothing must be absent, not zero")
	}
	if r := Rate(1, 3); *r != 0.3333 {
		t.Fatalf("Rate(1, 3) = %v", *r)
	}
	if rate(nil) != "" || rate(Rate(2, 3)) != "0.6667" {
		t.Fatal("rate cells")
	}
}

func TestWriteMakesTheDataset(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "2026-09")
	files, err := Build(fixture()).Write(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 8 {
		t.Fatalf("files = %v", files)
	}
	want := map[string]string{
		OutcomesFile:  "outcome,endpoints,share\nreport,2,0.5000\nerror,1,0.2500\ntimeout,1,0.2500\n",
		PhasesFile:    "phase,endpoints,pass,warn,fail,info,skip,fail_rate\nnet,2,2,0,0,0,0,0.0000\nprotocol,2,0,0,1,0,1,1.0000\n",
		ChecksFile:    "check_id,phase,endpoints,pass,warn,fail,info,skip,fail_rate\nnet.dns,net,2,2,0,0,0,0,0.0000\nprotocol.origin,protocol,2,0,0,1,0,1,1.0000\n",
		TransportFile: "transport,endpoints,reports,errors,timeouts,blocked,with_failure,failure_rate\nsse,1,0,1,0,0,0,\nstreamable-http,3,2,0,1,1,1,0.5000\n",
		AuthFile:      "auth_scheme,endpoints,reports,errors,timeouts,blocked,with_failure,failure_rate\nnone,1,1,0,0,0,1,1.0000\noauth,1,1,0,0,1,0,0.0000\n",
		ListingFile: "stage,reason,count\nentries,,10\ncurrent,,6\nendpoints,,5\nskipped,requires headers the user supplies,1\n" +
			"skipped,templated URL,2\nduplicates,,1\nexcluded,,0\nselected,,4\nchecked,,4\n",
	}
	for name, body := range want {
		got, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != body {
			t.Errorf("%s\n got:\n%s\nwant:\n%s", name, got, body)
		}
	}
}

// Nothing published may name, locate or quote an endpoint.
func TestNoPublishedFileIdentifiesAServer(t *testing.T) {
	dir := t.TempDir()
	files, err := Build(fixture()).Write(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		b, err := os.ReadFile(filepath.Join(dir, f))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(b), "secret") {
			t.Errorf("%s names an endpoint:\n%s", f, b)
		}
	}
}

func TestDataPackageDescribesEveryTable(t *testing.T) {
	dir := t.TempDir()
	if _, err := Build(fixture()).Write(dir); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, DataPackageFile))
	if err != nil {
		t.Fatal(err)
	}
	var p dataPackage
	if err := json.Unmarshal(b, &p); err != nil {
		t.Fatal(err)
	}
	if p.Name != "passmcp-census-2026-09" || p.Licenses[0].Name != "CC-BY-4.0" || p.Created != "2026-09-29T12:00:00Z" {
		t.Fatalf("descriptor = %+v", p)
	}
	if len(p.Resources) != 7 {
		t.Fatalf("%d resources", len(p.Resources))
	}
	for _, r := range p.Resources[:6] {
		if _, err := os.Stat(filepath.Join(dir, r.Path)); err != nil || r.Schema == nil {
			t.Fatalf("resource %+v", r)
		}
		for _, f := range r.Schema.Fields {
			if f.Name == "fail_rate" && f.Type != "number" || f.Name == "endpoints" && f.Type != "integer" {
				t.Fatalf("%s: field %+v", r.Name, f)
			}
		}
	}
}

func TestWriteReportsAnUnwritableDirectory(t *testing.T) {
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Build(fixture()).Write(filepath.Join(file, "sub")); err == nil {
		t.Fatal("wrote under a file")
	}
}
