<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Architecture Decision Records

Decisions about this repository that will be questioned later, with the
reasoning that produced them. Records are immutable once merged; a decision
that changes gets a new record superseding the old one. All four were made
with the first release.

| # | Decision | Status |
|---|---|---|
| [0001](0001-run-passmcp-as-a-program.md) | Run a released passmcp program; do not link its engine | Accepted |
| [0002](0002-non-invoking-phases-without-credentials.md) | Check only with phases that invoke nothing, and without credentials | Accepted |
| [0003](0003-editions-run-by-hand.md) | Every edition is started by hand; CI never runs one | Accepted |
| [0004](0004-aggregate-only.md) | Publish aggregate, one-dimensional tables only | Accepted |
