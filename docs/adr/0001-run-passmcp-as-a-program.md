<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# 0001. Run a released passmcp program; do not link its engine

## Status

Accepted.

## Context

A census figure is only as meaningful as the statement of what produced
it. passmcp's engine is GPL-3.0-only and internal; this repository's code
is Apache-2.0 and its data CC-BY-4.0.

## Decision

The census runs the passmcp binary installed from a release
(`go install satellion.com/passmcp/cmd/passmcp@v0.0.2`, `make passmcp`) and
reads its JSON report. The Makefile pins the release; the edition records
the version the binary reported.

## Consequences

- An edition's method is exactly a passmcp release plus the flags in
  `runner.Passmcp.Args`, which anyone can rerun.
- The census depends on the report format, and refuses a schema version
  other than 1 rather than miscounting it.
