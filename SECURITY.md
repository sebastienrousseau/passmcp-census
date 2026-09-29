<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Security Policy

passmcp-census checks other people's MCP servers and publishes aggregate
figures about them. Its security posture has two sides:

- **What a run does to the servers it checks.** It reads only, without
  credentials, only the servers the registry lists, and slowly.
- **What it publishes.** Counts and rates, never a server's name, URL or
  host, so no published figure points at a vulnerable server.

## Reporting a Vulnerability

Report security issues through [GitHub's private vulnerability reporting](https://github.com/sebastienrousseau/passmcp-census/security/advisories/new). Do not open a public issue.

You will receive an acknowledgement within **72 hours**. A confirmed
vulnerability is fixed within **90 days** of the report, or sooner when a
fix is straightforward. If the window cannot be met, you will be told why
and given a revised date.

Each of these is a vulnerability:

- a way to make a run contact a host the registry does not list;
- a way to make it send a credential or invoke a tool;
- a way to make a published file identify a server.

## Supported Versions

Only the latest release is supported.

## Security Measures

Each item names the test that enforces it.

- **Only non-invoking phases, without credentials.** passmcp runs with the
  phases net, discovery, handshake, protocol and catalog, `--auth none`, an
  empty configuration file, and no `PASSMCP_` environment variable.
  `TestArgsAreTheMethod`, `TestCheckRunsWithoutTheOperatorsCredentials`.
- **Polite by construction.** At most one check per host per interval, a
  deadline per endpoint, and `--rps` refused above 2.
  `TestRunSpacesChecksAgainstOneHost`, `TestRunAppliesTheDeadlinePerEndpoint`,
  `TestUsageErrors`.
- **Nothing published identifies a server.** Tables are one-dimensional
  counts, and server-chosen text never becomes a check id.
  `TestNoPublishedFileIdentifiesAServer`,
  `TestSummariseReducesEachCheckToItsWorstStatus`.
- **Supply chain.** No dependencies outside the Go standard library. CI
  runs `govulncheck` on every push, and passmcp is pinned to a release.
