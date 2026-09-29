<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Architecture

passmcp-census is one command with three subcommands over six internal
packages, and no dependency outside the Go standard library.

```text
list:       registry ──► selection ──► listing.json
run:        registry ──► selection ──► runner ──► passmcp (per endpoint) ──► record ──► records.ndjson, run.json
aggregate:  run.json + records.ndjson ──► aggregate ──► data/<edition>/*.csv, census.json, datapackage.json
```

| Package | Owns |
|---|---|
| `cmd/passmcp-census` | Flags, the working directory, the selection counts, the run metadata; JSON to stdout, progress to stderr |
| `internal/registry` | Paging the v0 servers API with retries, both entry shapes, the skip reasons, distinct URLs |
| `internal/exclude` | The exclusion list, in passmcp-registry's opt-out format |
| `internal/limiter` | The per-host start interval |
| `internal/runner` | The passmcp command line, its environment, workers and the per-endpoint deadline |
| `internal/record` | Reducing a passmcp JSON report to per-phase and per-check statuses |
| `internal/edition` | Edition ids, working-file names and the run metadata type |
| `internal/aggregate` | The tables, their CSV and JSON forms and the Data Package descriptor |

## Boundaries

- **passmcp is a program, not a library** ([ADR 0001](adr/0001-run-passmcp-as-a-program.md)).
  The census depends on passmcp's JSON report format (schema version 1),
  and refuses any other.
- **Records name endpoints; the dataset does not**
  ([ADR 0004](adr/0004-aggregate-only.md)). Only `aggregate` writes to
  `data/`, and its tables carry no identity.
- **Server text is untrusted.** Check ids and phase names pass through
  `record.Clean`; passmcp's error text is bounded to 400 bytes and stays
  in the private records.
