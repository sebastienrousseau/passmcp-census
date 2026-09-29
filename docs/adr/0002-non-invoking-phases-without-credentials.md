<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# 0002. Check only with phases that invoke nothing, and without credentials

## Status

Accepted.

## Context

The servers are not ours. Their owners have agreed to nothing beyond what
an anonymous client of a public listing does.

## Decision

passmcp runs the phases net, discovery, handshake, protocol and catalog —
the set passmcp-registry runs — with `--auth none`, an empty configuration
file and no `PASSMCP_` environment variable. Checks on one host start at
least ten seconds apart, passmcp makes at most one request a second within
a check, and each endpoint has a deadline.

## Consequences

- The census describes what an anonymous client sees. Execution,
  performance, resilience and auth findings are out of its scope.
- Widening any of this is a methodology change that needs an issue first.
  `TestArgsAreTheMethod` and `TestCheckRunsWithoutTheOperatorsCredentials`
  hold the line.
