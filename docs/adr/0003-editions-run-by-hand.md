<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# 0003. Every edition is started by hand; CI never runs one

## Status

Accepted.

## Context

An edition contacts every remote server the registry lists, which is a
decision about other people's infrastructure, and one a person should take
and review each time. `make census-list` shows how many endpoints that is
before any is contacted.

## Decision

There is no scheduled or CI-triggered run. `make census EDITION=<id>` is
the one way to produce an edition, run by the Maintainer, whose output is
reviewed before `data/<edition>/` is committed. Tests use fixtures and
loopback fakes only.

## Consequences

- The census is as fresh as the last edition the Maintainer ran. The
  ecosystem map's archive criterion applies: if editions stop, the
  repository is archived rather than left presenting stale data as current.
- A partial run is published as partial, never completed by extrapolation.
