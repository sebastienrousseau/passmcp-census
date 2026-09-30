<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Development

The single entry point for working on passmcp-census: toolchain, how to
reproduce every CI gate locally, how an edition is run, and how a release
is cut.

## Requirements

| Tool | Version | Why |
|---|---|---|
| Go | 1.26.8 or later, the `go` directive in `go.mod` | `GOTOOLCHAIN=auto` downloads it; CI tests on that version and on latest stable |
| make | any | Task runner for everything below |
| passmcp | the release `PASSMCP_VERSION` in the Makefile pins | Only for a census run; `make passmcp` installs `satellion.com/passmcp/cmd/passmcp@v0.0.4` into `build/bin` |

Optional, only for the gate that uses it: `golangci-lint` v2 (`make lint`),
`reuse` (`make reuse`), Python 3.12 with the hash-locked
`docs/requirements.txt` (`make manual`), `markdownlint-cli2`, `codespell`
and `lychee` (the Docs Lint workflow and `pre-commit`).

The Go floor is raised only when a release needs a language feature, and
the changelog says so.

## Reproducing every CI gate

| CI job | Local command |
|---|---|
| Test (three OSes × two Go versions) | `make test` |
| Race & Shuffled Tests | `make test-race` |
| Coverage Gate (85% per package) | `make coverage` |
| Lint, with the complexity ceilings | `gofmt -l .` and `make lint` |
| Vulnerability Scan | `go run golang.org/x/vuln/cmd/govulncheck@latest ./...` |
| Repository Checks | `make spdx-check name-guard readme-check versions reuse` |
| Licence Headers | `make spdx-check` |
| Markdown & Spelling | `make readme-check`, `markdownlint-cli2 '**/*.md'` and `codespell` |
| Link Check | `lychee --offline --include-fragments '**/*.md'` |
| DCO check | `git log --format=%B origin/main.. \| grep Signed-off-by` |
| Manual (build, and on `main` deploy with `coverage.json`) | `make manual` and `make coverage-json` |
| CodeQL | not reproducible locally; runs on push, pull request and weekly |
| OpenSSF Scorecard | not reproducible locally; runs on push to `main` and weekly, and publishes to [scorecard.dev](https://scorecard.dev/viewer/?uri=github.com/sebastienrousseau/passmcp-census) |

`make` with no target runs the gates that need no network, cheap ones
first.

## Complexity

`make lint` enforces the portfolio's per-function ceilings through
golangci-lint: cyclomatic complexity 10 (`gocyclo`), cognitive complexity
15 (`gocognit`) and 60 lines (`funlen`). Every function is under them, so
there is no baseline of existing offenders. Halstead difficulty has no
golangci-lint analyser and is not gated.

## Coverage

The gate is 85% statement coverage in every package with statements.
`make coverage` writes `coverage.out`; ci.yml's Coverage Gate checks each
package against the threshold.

The README's coverage badge is the whole-module number: `make coverage-json`
turns the profile into a
[shields.io endpoint document](https://shields.io/badges/endpoint-badge),
`build/coverage.json`, brightgreen from 90%, green from 85%, yellow from
70% and red below. The Manual workflow publishes it on every push to
`main` at <https://sebastienrousseau.com/passmcp-census/coverage.json>,
which the badge reads.

## Running an edition

An edition contacts every remote server the registry lists. It is started
by the Maintainer by hand, never by CI
([ADR 0003](docs/adr/0003-editions-run-by-hand.md)).

```sh
make census-list EDITION=2026-09   # the registry snapshot only; no server contacted
make census EDITION=2026-09        # passmcp at the pinned release, then aggregate
```

`make census` writes the private working files (listing, per-endpoint
records, run metadata) to `build/census/<edition>/` and the dataset to
`data/<edition>/`. Review the dataset, then commit only `data/<edition>/`.

`make demo` runs an edition against loopback only and records it as the
README demo, `.github/demo.gif`: `.github/demo/registry` is served as a fake
registry with `python3 -m http.server`, listing three of passmcp's example
servers, and [VHS](https://github.com/charmbracelet/vhs) (`vhs`, `ttyd`,
`ffmpeg`) records `.github/demo.tape` in `build/demo/work`, so nothing is
written to `data/`. Regenerate it when what `run` or `aggregate` prints
changes, and leave 90 seconds between renders: the servers a render starts
stop themselves then.

## Test layout

Each package's tests sit beside it. The registry tests serve fixture pages
on loopback; the runner and command tests re-execute the test binary as a
fake passmcp, so they run on every OS; the record tests read a real passmcp
0.0.1 report, trimmed, and a synthetic failing one, from
`internal/record/testdata/`.

## Release model

The family moves in lockstep: every repository is released at passmcp's
version, on a `feat/vX.Y.Z` branch.

1. Date the `## [X.Y.Z]` heading in `CHANGELOG.md`, and write
   `docs/releases/vX.Y.Z.md` with a `## Highlights ⭐️` section.
2. Update the version in the README's ecosystem sentence and
   `CITATION.cff`. Move `PASSMCP_VERSION` in the Makefile, and every
   document that names it, to the newest passmcp release that is
   tagged; it may trail this release, never lead it.
3. `make versions`, and optionally the Release workflow's dry run.
4. Push a signed annotated tag `vX.Y.Z` with the message
   `passmcp-census vX.Y.Z`. The Release workflow publishes the release page.
5. Read the tag and the release page back before calling it done.

## Conventions

- Stdout carries a command's JSON result; every diagnostic goes to stderr.
- Every exported identifier is documented.
- Anything the registry or a server returns is untrusted input, bounded
  before it reaches a record, and never reaches the dataset as text.
