<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Disclosure log

## Policy

The census publishes aggregate figures only: counts and rates by outcome,
check, phase, transport and authorization scheme. No published file names,
locates or quotes a server, so no figure points an attacker at a vulnerable
one, and nothing needs holding back for an owner under an embargo. A
finding about a specific server is not a census output; the per-server
scorecard, with its withholding rules and 90-day window, is
[passmcp-registry](https://github.com/sebastienrousseau/passmcp-registry/blob/main/DISCLOSURE.md).

The per-endpoint working files a run writes stay on the machine that ran
it, under `build/census/`, which git ignores. They are not published.

## Asking to be excluded

If you own a server listed in the MCP Registry and don't want future
editions to contact it, open an issue titled `exclude: <registry name>`
from the account or organisation that owns the namespace, or email the
address in [SECURITY.md](https://github.com/sebastienrousseau/passmcp-census/blob/main/SECURITY.md). A
namespace followed by `/*` covers every server under it. No reason is
needed. The name is added to
[`data/exclusions.txt`](https://github.com/sebastienrousseau/passmcp-census/blob/main/data/exclusions.txt)
before the next edition, and recorded below.

## Exclusions

| Date | Entry | Requested through |
|---|---|---|
| — | none received | — |

## Log

| Date | Event |
|---|---|
| — | No edition has been run. |
