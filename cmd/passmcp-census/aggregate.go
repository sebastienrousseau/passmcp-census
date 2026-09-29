// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"satellion.com/passmcp-census/internal/aggregate"
	"satellion.com/passmcp-census/internal/edition"
	"satellion.com/passmcp-census/internal/record"
)

// maxRecordLine bounds one line of records.ndjson.
const maxRecordLine = 4 << 20

func cmdAggregate(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("aggregate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	id := fs.String("edition", "", "edition id, YYYY-MM (required)")
	work := fs.String("work", "", "the run's working directory (default build/census/<edition>)")
	out := fs.String("out", "", "where the dataset is written (default data/<edition>)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if err := edition.Valid(*id); err != nil {
		_, _ = fmt.Fprintln(stderr, "passmcp-census aggregate:", err)
		return 2
	}
	if *work == "" {
		*work = filepath.Join("build", "census", *id)
	}
	if *out == "" {
		*out = filepath.Join("data", *id)
	}
	d, err := load(*id, *work)
	if err != nil {
		return fail(stderr, err)
	}
	files, err := d.Write(*out)
	if err != nil {
		return fail(stderr, err)
	}
	b, _ := json.MarshalIndent(map[string]any{"edition": *id, "dir": *out, "files": files, "totals": d.Totals}, "", "  ")
	_, _ = fmt.Fprintln(stdout, string(b))
	return 0
}

// load reads a run's metadata and records and aggregates them.
func load(id, work string) (aggregate.Dataset, error) {
	var meta edition.Run
	b, err := os.ReadFile(filepath.Join(work, edition.RunFile)) // #nosec G304 -- the operator's own working directory
	if err != nil {
		return aggregate.Dataset{}, err
	}
	if err := json.Unmarshal(b, &meta); err != nil {
		return aggregate.Dataset{}, fmt.Errorf("%s: %w", edition.RunFile, err)
	}
	if meta.Edition != id {
		return aggregate.Dataset{}, fmt.Errorf("%s is edition %q, not %q", work, meta.Edition, id)
	}
	recs, err := readRecords(filepath.Join(work, edition.RecordsFile))
	if err != nil {
		return aggregate.Dataset{}, err
	}
	if len(recs) != meta.Listing.Checked {
		return aggregate.Dataset{}, fmt.Errorf("%s holds %d records; the run recorded %d", edition.RecordsFile, len(recs), meta.Listing.Checked)
	}
	return aggregate.Build(meta, recs), nil
}

func readRecords(path string) ([]record.Record, error) {
	f, err := os.Open(path) // #nosec G304 -- the operator's own working directory
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64<<10), maxRecordLine)
	var recs []record.Record
	for n := 1; sc.Scan(); n++ {
		var r record.Record
		if err := json.Unmarshal(sc.Bytes(), &r); err != nil {
			return nil, fmt.Errorf("%s line %d: %w", edition.RecordsFile, n, err)
		}
		recs = append(recs, r)
	}
	return recs, sc.Err()
}
