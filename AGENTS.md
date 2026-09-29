<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Working on passmcp-census as an AI agent

passmcp-census publishes the passmcp reliability census: aggregate figures
about the MCP Registry's remote servers, the methodology behind them, the
disclosure log and the command that reproduces them. These are the
invariants for AI-assisted contributions. Read
[DEVELOPMENT.md](DEVELOPMENT.md) for the toolchain.

## Hard gates

| Gate | Command |
|---|---|
| 85% statement coverage, every package | `make coverage` (ci.yml applies it per package) |
| Race detector, randomised order | `make test-race` |
| Lint at zero findings, complexity ceilings included | `make lint` |
| SPDX header on every source file, REUSE compliant | `make spdx-check reuse` |
| README follows the template | `make readme-check` |
| Every version-bearing file agrees | `make versions` |
| The retired product name appears nowhere | `make name-guard` |

## Things that are load-bearing

- **Never contact a real server or the real registry from a test.** Use
  the fixtures and loopback fakes. A census run is the Maintainer's
  decision (`make census`), never CI's and never an agent's.
- **Never fabricate, extrapolate or backfill a figure.** A dataset under
  `data/<edition>/` is exactly what `passmcp-census aggregate` wrote from
  one run. A partial run is published as partial (`complete: false`).
- **Nothing published identifies a server.** Tables stay one-dimensional
  counts; no name, URL, host or server-chosen text reaches `data/`.
- **Only non-invoking phases, without credentials.** `runner.Phases`,
  `--auth none`, an empty passmcp configuration, no `PASSMCP_` variables.
  Widening any of it is a methodology change that needs an issue first.
- **The working directory is private.** `build/census/` names endpoints;
  it is ignored by git and never published.

## Commits

Signed, with a DCO sign-off added by the human, and a Conventional Commits
subject of 50 characters or fewer. Never rewrite pushed history.
