<!-- SPDX-License-Identifier: Apache-2.0 -->
<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->

<p align="center">
  <img src="https://raw.githubusercontent.com/sebastienrousseau/passmcp/main/.github/logo.svg" alt="passmcp-census logo" width="128" />
</p>

<h1 align="center">passmcp-census</h1>

<p align="center">
  The passmcp reliability census: aggregate figures about the MCP Registry's remote servers, each checked by a released passmcp read-only and without credentials, with the methodology, the disclosure log and the command that reproduces them.
</p>

<p align="center">
  <a href="https://github.com/sebastienrousseau/passmcp-census/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/sebastienrousseau/passmcp-census/ci.yml?branch=main&style=for-the-badge&logo=github&label=Build" alt="Build" /></a>
  <a href="https://github.com/sebastienrousseau/passmcp-census/blob/main/DEVELOPMENT.md#coverage"><img src="https://img.shields.io/endpoint?url=https%3A%2F%2Fsebastienrousseau.com%2Fpassmcp-census%2Fcoverage.json&style=for-the-badge&logo=codecov&logoColor=white" alt="Coverage" /></a>
  <a href="https://github.com/sebastienrousseau/passmcp-census/releases"><img src="https://img.shields.io/github/v/release/sebastienrousseau/passmcp-census?style=for-the-badge&color=fc8d62&logo=github&label=Release" alt="Release" /></a>
  <a href="https://sebastienrousseau.com/passmcp-census/"><img src="https://img.shields.io/badge/docs-manual-007d9c?style=for-the-badge&labelColor=555555&logo=readthedocs&logoColor=white" alt="Docs" /></a>
  <a href="https://scorecard.dev/viewer/?uri=github.com/sebastienrousseau/passmcp-census"><img src="https://img.shields.io/ossf-scorecard/github.com/sebastienrousseau/passmcp-census?style=for-the-badge&label=OpenSSF%20Scorecard&logo=openssf" alt="OpenSSF Scorecard" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-Apache--2.0-blue.svg?style=for-the-badge" alt="License: Apache-2.0" /></a>
  <a href="https://github.com/sebastienrousseau/passmcp-census/blob/main/DEVELOPMENT.md#requirements"><img src="https://img.shields.io/badge/go-1.26.8%2B-93450a.svg?style=for-the-badge&logo=go" alt="Go 1.26.8+" /></a>
</p>

---

## Contents

**Getting started**

- [Install](#install) — build from source, and the pinned passmcp
- [Requirements](#requirements) — Go, passmcp, and the network for a run
- [Quick Start](#quick-start) — list the registry, then build the tooling's tests

**The passmcp-census ecosystem**

- [The passmcp-census ecosystem](#the-passmcp-census-ecosystem) — `passmcp`, `passmcp-reporting`, `passmcp-server`, `passmcp-action`, `passmcp-graph`, `passmcp-registry`, `passmcp-lsp`, `passmcp-census`, `satellion.com`

**Library reference**

- [Capabilities at a glance](#capabilities-at-a-glance) — the current surface by theme
- [Ecosystem comparison](#ecosystem-comparison) — short matrix
- [Benchmarks](#benchmarks) — why there are none
- [Features](#features) — selection, politeness, aggregate-only output
- [Configuration](#configuration) — every flag
- [Examples](#examples) — a run's output and the dataset files

**Operational**

- [When not to use passmcp-census](#when-not-to-use-passmcp-census) — limitations
- [Development](#development) — make targets and CI
- [Security](#security) — what a run can and cannot do
- [Documentation](#documentation) — all reference docs
- [Stability guarantees](#stability-guarantees) — the dataset format and the command
- [License](#license)

---

## Install

### As a Go program, from source

```sh
git clone https://github.com/sebastienrousseau/passmcp-census
cd passmcp-census
make build          # build/passmcp-census
make passmcp        # passmcp at the pinned release, into build/bin
```

`make passmcp` runs:

```sh
go install satellion.com/passmcp/cmd/passmcp@v0.0.3
```

with `GOBIN` set to `build/bin`. passmcp-census has no dependency outside
the Go standard library. There are no release binaries; the command
installs from the module proxy.

---

## Requirements

| Requirement | Floor | Enforced by |
|---|---|---|
| Go | the `go` directive in [`go.mod`](go.mod), 1.26.8 | CI tests on that version and on latest stable, on Linux, macOS and Windows |
| passmcp | the release the Makefile pins, v0.0.3 | `make passmcp` installs exactly that release; each edition records the version the binary reported |
| Network | HTTPS to the registry for `list`; to every listed server for `run` | nothing in CI or the tests leaves loopback |

The Go floor is raised only when a release needs a language feature, and
the changelog says so.

---

## Quick Start

```sh
make build
build/passmcp-census list --edition 2026-09
```

`list` reads the official MCP Registry and prints, as JSON on stdout, what a
run would check: entries read, current servers, remote endpoints, how many
were skipped and why, duplicates, exclusions and the final selection. It
contacts no MCP server; page-by-page progress goes to stderr. It writes the
listing to `build/census/2026-09/listing.json`, which names servers and is
never published.

A full edition — every selected endpoint checked, then aggregated into
`data/2026-09/` — is `make census EDITION=2026-09`. It is the Maintainer's
decision to run one ([ADR 0003](docs/adr/0003-editions-run-by-hand.md)).

---

## The passmcp-census ecosystem

Every component is released at **0.0.3** and moves in lockstep: one version across the family, released together ([docs/ecosystem.md](https://github.com/sebastienrousseau/passmcp/blob/main/docs/ecosystem.md)).

| Component | Purpose | Use case |
| :--- | :--- | :--- |
| [passmcp](https://github.com/sebastienrousseau/passmcp) | The MCP server diagnostic: checks in nine phases, every finding tied to the request that showed it, signed attestations | Test a server before your agents trust it, and gate it in CI |
| [passmcp-reporting](https://github.com/sebastienrousseau/passmcp-reporting) | The attestation format, its JSON Schemas and offline verifier, the graph model, and the agentgateway processor | Verify an attestation in a gateway, registry or pipeline |
| [passmcp-server](https://github.com/sebastienrousseau/passmcp-server) | passmcp's diagnostics as read-only MCP tools | Evaluate a server, or check an attestation, from inside the agent |
| [passmcp-action](https://github.com/sebastienrousseau/passmcp-action) | passmcp in GitHub Actions and GitLab CI, the image pinned by digest | Fail a build on the findings you choose |
| [passmcp-graph](https://github.com/sebastienrousseau/passmcp-graph) | A local graph of agents, servers, tools and identities built from attestations | Find inherited risk and over-privilege, and gate on policy |
| [passmcp-registry](https://github.com/sebastienrousseau/passmcp-registry) | A signed public scorecard of the MCP Registry's remote servers | Check a public server's standing before connecting to it |
| [passmcp-lsp](https://github.com/sebastienrousseau/passmcp-lsp) | A language server for MCP artefacts, with check-id hover from the guidance catalogue | Catch mistakes in server.json, tool schemas and client configuration while editing |
| [passmcp-census](https://github.com/sebastienrousseau/passmcp-census) | The published reliability census: dataset, methodology, disclosure log and reproduction command | Cite ecosystem-wide reliability figures, and reproduce them |
| [satellion.com](https://github.com/sebastienrousseau/satellion.github.io) | The website, the Go module paths and the format URIs | Read the manual, and resolve `satellion.com/...` imports |

This repository is the census: the tooling that produces an edition, the
methodology that says what it means, and the editions themselves under
`data/`. passmcp-registry publishes per-server records; the census
publishes only figures across all of them.

---

## Capabilities at a glance

| Area | Capability | Status |
| :--- | :--- | :--- |
| Selection | `passmcp-census list`: the registry snapshot and what a run would check | [Released in 0.0.2](https://github.com/sebastienrousseau/passmcp-census/releases/tag/v0.0.2) |
| Run | `passmcp-census run`: each endpoint checked by passmcp v0.0.3, read-only, without credentials, rate-limited | [Released in 0.0.2](https://github.com/sebastienrousseau/passmcp-census/releases/tag/v0.0.2) |
| Dataset | `passmcp-census aggregate`: CSV tables, `census.json`, `datapackage.json` | [Released in 0.0.2](https://github.com/sebastienrousseau/passmcp-census/releases/tag/v0.0.2) |
| Documentation | Methodology, disclosure log, decision records, rendered manual | [Released in 0.0.2](https://github.com/sebastienrousseau/passmcp-census/releases/tag/v0.0.2) |
| Edition 2026-09 | The first published figures | Not yet run: no figures exist ([docs/index.md](docs/index.md#editions)) |

---

## Ecosystem comparison

Two repositories in the family check the registry's remote servers with
the same selection and the same non-invoking phases. They differ in what
they publish.

| Project | Unit published | Identifies servers | Disclosure handling |
| :--- | :---: | :---: | :---: |
| **passmcp-census** | aggregate counts and rates per edition | no | none needed; exclusions logged |
| [passmcp-registry](https://github.com/sebastienrousseau/passmcp-registry) | one signed record per server | yes | withholding rules, 90-day window |
| `passmcp check` by hand | one report | yes, to whoever runs it | the operator's |

The comparison is between repositories of this family only; no external
project is claimed to be comparable.

---

## Benchmarks

There are no benchmarks. A run's duration is set by its politeness
settings and by the servers it checks, not by this code, and each edition
records its own start, finish and duration in its metadata.

| Scenario | Result | Environment |
| :--- | ---: | :--- |
| Edition run time | recorded per edition in `census.json` (`run.duration_seconds`) | the machine and network that ran it |

---

## Features

**The selection passmcp-registry documents.** The latest version of every
active server, its `streamable-http` and `sse` endpoints, less templated
URLs, URLs carrying credentials and endpoints that need user-supplied
headers; each URL once; excluded servers dropped. Every step is counted.

**Nothing that invokes, nothing that authenticates.** Phases net,
discovery, handshake, protocol and catalog only; `--auth none`; an empty
passmcp configuration; no `PASSMCP_` variable reaches passmcp.

**Polite by construction.** Four endpoints at once, ten seconds between
checks on one host, one request a second within a check, a three-minute
deadline per endpoint, and retries with backoff when the registry falters.

**Aggregate only.** Six one-dimensional tables, `census.json` and a
Frictionless `datapackage.json`, CC-BY-4.0. No name, URL, host or
server-chosen text is published.

**Honest about gaps.** A limited or interrupted run is marked
`complete: false` and exits non-zero; a rate over nothing is empty, not
zero.

---

## Configuration

`run` and `list` share these flags:

| Flag | Default | Meaning |
|---|---|---|
| `--edition` | required | Edition id, `YYYY-MM` |
| `--registry` | `https://registry.modelcontextprotocol.io` | MCP Registry base URL |
| `--work` | `build/census/<edition>` | Private working directory |
| `--exclusions` | `data/exclusions.txt` | Servers whose owners asked not to be contacted |
| `--passmcp` | `passmcp` | passmcp binary (`run` only) |
| `--workers` | `4` | Endpoints checked at once |
| `--host-interval` | `10s` | Minimum time between checks starting on one host |
| `--timeout` | `3m0s` | Deadline for one endpoint |
| `--call-timeout` | `20s` | passmcp's per-call timeout |
| `--rps` | `1` | passmcp's requests per second within a check; above 2 is refused |
| `--limit` | `0` (all) | Check at most this many; the run is then marked incomplete |

`aggregate` takes `--edition`, `--work` (default `build/census/<edition>`)
and `--out` (default `data/<edition>`). Nothing is read from the
environment.

---

## Examples

The command's own help:

```sh
build/passmcp-census help
```

A run prints its metadata to stdout as JSON — edition, census and passmcp
versions, registry snapshot times, run times and duration, parameters,
listing counts, the passmcp command line and the reproduction command —
and one progress line per endpoint to stderr. `aggregate` writes:

```text
data/<edition>/outcomes.csv      report, error, timeout
data/<edition>/listing.csv       each selection stage and skip reason
data/<edition>/by-phase.csv      each phase, by status
data/<edition>/by-check.csv      each check id, by status
data/<edition>/by-transport.csv  streamable-http, sse
data/<edition>/by-auth.csv       none, oauth, other, unreached
data/<edition>/census.json       every table and the run's metadata
data/<edition>/datapackage.json  the Frictionless descriptor
```

The column definitions are in [docs/methodology.md](docs/methodology.md#tables).

---

## When not to use passmcp-census

- **To judge one server.** The census publishes no per-server result. Use
  [passmcp-registry](https://github.com/sebastienrousseau/passmcp-registry)
  or run `passmcp check` yourself.
- **For servers that need credentials or headers.** They are skipped and
  counted; the census shows only what an anonymous client sees.
- **For tool behaviour, performance or resilience.** Those phases call
  tools or hold sessions open, and the census does not run them.
- **For stdio servers.** Only remote HTTP endpoints are listed and checked.
- **As current data between editions.** An edition describes the day it
  ran; none has been published yet.

---

## Development

```bash
make            # format, vet, lint, headers, README, name, versions, tests
make test-race  # race detector, randomised order
make coverage-json  # build/coverage.json, the document behind the badge
make manual     # the rendered manual, strictly
make census-list EDITION=2026-09  # registry snapshot only
```

Every gate CI runs has a local form; [DEVELOPMENT.md](DEVELOPMENT.md) maps
them. The tests use fixture registry pages on loopback and the test binary
as a fake passmcp; none contacts the real registry or a real server.

---

## Security

A run passes passmcp only the flags in
[docs/methodology.md](docs/methodology.md#what-runs), with an empty
configuration and no `PASSMCP_` variables, and runs no phase that calls a
tool or presents a credential; each property is a named test. The
published tables are one-dimensional counts, and a test fails the build if
a published file names an endpoint. The module has no dependencies; CI
runs `govulncheck` on every push.

Report vulnerabilities according to [`SECURITY.md`](SECURITY.md).

---

## Documentation

The four entry points, identical across every repo in the family:

- **[User Manual](https://sebastienrousseau.com/passmcp-census/)** — the methodology, the disclosure log and the decision records, rendered
- **[API reference](https://pkg.go.dev/satellion.com/passmcp-census)** — this module; its packages are internal, so it documents the command
- **[Developer docs](DEVELOPMENT.md)** — toolchain, task map, reproducing every CI gate locally
- **[Ecosystem map](https://github.com/sebastienrousseau/passmcp/blob/main/docs/ecosystem.md)** — the family, the published artefacts, the lockstep version rule

| Document | Covers |
|---|---|
| [`docs/methodology.md`](docs/methodology.md) | Selection, what runs, reduction, tables, metadata |
| [`docs/disclosure-log.md`](docs/disclosure-log.md) | The aggregate-only policy and every exclusion |
| [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) | Packages and how a run flows |
| [`docs/adr/`](docs/adr/README.md) | Decision records for this repository |
| [`docs/releases/`](docs/releases/v0.0.3.md) | Release highlights, one file per release |
| [`SECURITY.md`](SECURITY.md) | Disclosure policy, supported versions, what is guaranteed |
| [`CONTRIBUTING.md`](CONTRIBUTING.md) | Signed-commit and DCO policy, what a change needs |
| [`CHANGELOG.md`](CHANGELOG.md) | Per-release notes |
| [`SUPPORT.md`](SUPPORT.md) | Where to ask, and what to expect |
| [`CITATION.cff`](CITATION.cff) | How to cite the census |

---

## Stability guarantees

passmcp-census is pre-1.0, carries the family's version, and follows
SemVer with the patch digit moving for everything until 1.0.

**The breaking axis is what a consumer of the data relies on.** These are
breaking:

- Removing or renaming a published file, a column or a `census.json` field
- Changing what a column counts, or how a rate is computed
- Changing a command, a flag or its default, or what goes to stdout

Adding a table, a column or a field is not breaking. A published edition is
never rewritten: a correction is a new edition, and the methodology
change behind it is in the CHANGELOG.

**Deprecation window.** A deprecated file, column or flag keeps working for
at least one release after the release that announces it.

---

## License

The code is licensed under the **[Apache License 2.0](LICENSE)**. The
census data under [`data/`](data/) is licensed under
**[CC-BY-4.0](LICENSES/CC-BY-4.0.txt)**: cite the census with
[`CITATION.cff`](CITATION.cff) and name the edition.
[`REUSE.toml`](REUSE.toml) records which licence covers which file.

<p align="right"><a href="#contents">Back to Top</a></p>
