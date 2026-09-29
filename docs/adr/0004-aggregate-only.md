<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# 0004. Publish aggregate, one-dimensional tables only

## Status

Accepted.

## Context

Per-server results can hand an attacker a working way into a server whose
owner has not heard about it; passmcp-registry handles that with
withholding rules and a 90-day window. A census exists to state
ecosystem-wide figures, which need no identities.

## Decision

The published dataset is counts and rates by outcome, check, phase,
transport and authorization scheme, each table on one attribute. No name,
URL, host or server-chosen text reaches `data/`. The per-endpoint records
stay in the private working directory.

## Consequences

- There is nothing to embargo; the disclosure log records the policy and
  every exclusion request instead of per-server notices.
- Cross-tabulations (a check by transport, say) are not published, because
  a small cell could narrow to one server. Adding one needs this record
  superseded.
- `TestNoPublishedFileIdentifiesAServer` holds the line.
