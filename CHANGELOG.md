<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Changelog

All notable changes are documented here, in the format of
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/). Versions move by
0.0.1 a release, in lockstep with the rest of the passmcp family.

## [0.0.2] — 2026-09-29

The first release. passmcp-census was never tagged at 0.0.1; it joins the
family at the family's current version.

### Added

- **`passmcp-census list`**: reads the MCP Registry's v0 servers API
  (`version=latest`) and selects the remote endpoints the way
  passmcp-registry does, less templated URLs, URLs carrying credentials,
  endpoints that need user-supplied headers, duplicate URLs and excluded
  servers. It contacts no MCP server.
- **`passmcp-census run`**: checks each selected endpoint with a released
  passmcp, phases net, discovery, handshake, protocol and catalog only,
  `--auth none`, an empty passmcp configuration and no `PASSMCP_`
  variables; bounded concurrency, ten seconds between checks on one host,
  one request a second within a check, and a deadline per endpoint.
- **`passmcp-census aggregate`**: turns a run into the published dataset —
  outcome, listing, phase, check, transport and authorization-scheme tables
  as CSV, `census.json` and a Frictionless `datapackage.json` — with no
  server identified.
- The methodology, the disclosure log, a rendered manual, decision records
  and the family's repository files.

### Changed

- **Method**: a census runs passmcp v0.0.2 (`PASSMCP_VERSION`), moved
  from v0.0.1 during development so the census requires the newest
  released passmcp. The flags the runner passes and the report fields it
  reads are unchanged in v0.0.2; its one engine change rewords the detail
  of `protocol.id_echo`, which the census does not read. No edition was
  run at v0.0.1 ([methodology](docs/methodology.md#method-history)).

No census edition is published in this release.
