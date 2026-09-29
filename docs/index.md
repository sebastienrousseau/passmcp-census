<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# passmcp-census documentation

The passmcp reliability census: aggregate figures about the remote servers
listed in the official MCP Registry, each checked by a released passmcp
read-only and without credentials, with the methodology, the disclosure
log and the command that reproduces them.

| Document | Covers |
|---|---|
| [README](https://github.com/sebastienrousseau/passmcp-census/blob/main/README.md) | Install, Quick Start, configuration, limitations |
| [Methodology](methodology.md) | What is selected, what is run, how a report is reduced, what each table means |
| [Disclosure log](disclosure-log.md) | The aggregate-only policy, and every exclusion request |
| [Architecture](ARCHITECTURE.md) | The packages and how a run flows |
| [Decision records](adr/README.md) | Decisions made in this repository, and why |
| [Release 0.0.2](releases/v0.0.2.md) | The highlights of the first release |

## Editions

| Edition | Status |
|---|---|
| 2026-09 | Not yet run. No figures are published for it. |

A published edition lives under
[`data/<edition>/`](https://github.com/sebastienrousseau/passmcp-census/tree/main/data)
and is CC-BY-4.0.

## Coverage

The README's coverage badge reads
[coverage.json](https://sebastienrousseau.com/passmcp-census/coverage.json),
published with this site by the Manual workflow.

What each passmcp check means, and how to fix it, is passmcp's manual:
<https://satellion.com/passmcp/docs/>.
