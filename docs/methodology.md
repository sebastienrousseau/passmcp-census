<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Methodology

This page states exactly what a census edition measures. A change to
anything here changes what an edition means, and is recorded in the
CHANGELOG.

## Selection

`passmcp-census list` (and the first step of `run`) reads the registry's
v0 servers API, 100 entries a page, asking for the latest version of each
server only (`version=latest`). Every entry is still checked client-side:
an entry counts as current when its official metadata says `isLatest` and
its status is `active` (or absent). This is the selection
[passmcp-registry](https://github.com/sebastienrousseau/passmcp-registry)
documents. From the current entries' remote endpoints, these are set aside,
and counted by reason in `listing.csv`:

| Reason | Why |
|---|---|
| not a remote HTTP transport | only `streamable-http` and `sse` are checked |
| templated URL | the user fills it in; there is nothing to connect to |
| not an http(s) URL | |
| URL carries credentials | never sent anywhere |
| requires headers the user supplies | an anonymous client cannot connect |

Then each URL is kept once, however many servers list it (`duplicates`),
and servers on the [exclusion list](disclosure-log.md#exclusions) are
dropped (`excluded`). The unit of the census is a distinct endpoint URL.

## What runs

Each endpoint is checked by the passmcp release pinned in the Makefile
(`PASSMCP_VERSION`, v0.0.2), with:

```sh
passmcp check <url> --phases net,discovery,handshake,protocol,catalog \
  --auth none --output json --no-color --log-level error --rps 1 --timeout 20s \
  --user-agent "passmcp-census/<version> (+https://github.com/sebastienrousseau/passmcp-census)"
```

passmcp runs with `PASSMCP_CONFIG` naming an empty configuration file and
with every `PASSMCP_` environment variable removed, so no profile or
exported secret can add a credential or allow a mutation.

- **No tool is called.** The execution, performance and resilience phases
  call tools or hold sessions open, and the auth phase presents a made-up
  credential; none of them runs.
- **Slowly.** Four endpoints are checked at once; checks against one host
  start at least ten seconds apart; passmcp makes at most one request a
  second within a check, and `run` refuses a rate above two.
- **Bounded.** Each endpoint has a three-minute deadline; one that runs
  out is recorded as a timeout.

## Reduction

passmcp's exit status 0 or 2 with a JSON report (schema version 1) is an
outcome of `report`; any other exit, or output that is not such a report,
is `error`; the deadline is `timeout`. A report is reduced to:

- one status per phase, as passmcp reports it;
- one status per check id: a check that produced several findings (one
  per tool, say) counts once, at its worst status, in the order
  fail > warn > pass > info > skip. A computed id (`auth.source.<field>`)
  counts as its family; an id not of passmcp's form counts as
  `unrecognised`, so server-chosen text never becomes a row;
- an authorization scheme, from what passmcp observed without credentials:
  `none` (the server did not ask), `oauth` (it asked and advertised OAuth
  metadata), `other` (it asked and advertised none), `unreached`;
- whether the run was blocked before the later phases.

## Tables

Every table is one-dimensional: no table crosses two attributes, so no row
can narrow to a single server.

| File | Rows | Rate |
|---|---|---|
| `outcomes.csv` | report, error, timeout | share of endpoints checked |
| `listing.csv` | each selection stage and skip reason | — |
| `by-phase.csv` | each phase, counting reports by status | `fail_rate` = fail / (pass + warn + fail) |
| `by-check.csv` | each check id, counting reports by status | as above |
| `by-transport.csv` | each transport, counting every endpoint by outcome | `failure_rate` = reports with any failing check / reports |
| `by-auth.csv` | each authorization scheme, counting reports | as above |

A rate over nothing is empty, not zero. Rates are rounded to four decimal
places. `census.json` holds every table and the run's metadata;
`datapackage.json` describes the files as a
[Frictionless Data Package](https://specs.frictionlessdata.io/data-package/).

## Metadata

Each edition records the passmcp version the binary reported, the registry
and its snapshot start and finish, the run's start, finish and duration,
every parameter above, the passmcp command line, whether the run was
complete, and the reproduction command (`make census EDITION=<edition>`).

## What a figure is not

- Not a score. passmcp's score weighs the phases a census does not run.
- Not a verdict about any server. It is how many endpoints showed a
  property to an anonymous client on the day of the run.
- Not comparable across passmcp versions without reading both methods.

## Method history

Each change to the pinned passmcp release or to anything above, with the
passmcp-census release that made it. The [CHANGELOG](https://github.com/sebastienrousseau/passmcp-census/blob/main/CHANGELOG.md)
records the same changes.

| Release | Change | Effect on the figures |
|---|---|---|
| 0.0.2 | passmcp pinned at v0.0.2, moved from v0.0.1 during development | None expected: v0.0.2 accepts the same flags and writes the same report fields (schema version 1). Its one engine change rewords the detail text of `protocol.id_echo`, which the census does not read; the check's status is unchanged. No edition was run at v0.0.1. |
