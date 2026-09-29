<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Contributing

passmcp-census publishes figures about servers other people run. A change
here is judged first by what it does to those servers and their owners,
and only then by what it adds.

## Before you start

- Open an issue for anything larger than a fix, so the methodology is
  agreed before the code.
- Read [AGENTS.md](AGENTS.md): it lists the rules that are expensive to
  discover from a diff.

## Making a change

1. Fork, branch from `main`, and keep the change to one concern.
2. Run the local gates:

   ```sh
   make            # format, vet, lint, headers, README, name, versions, tests
   make test-race  # race detector, randomised order
   ```

3. Commit with a DCO sign-off (`git commit -s`) and a signed commit, with a
   [Conventional Commits](https://www.conventionalcommits.org/) subject of
   50 characters or fewer.
4. Open a pull request against `main`.

## What a change needs

- **85% statement coverage in every package.**
- **Tests against fixtures only.** A test never contacts the real registry
  or a real server; use the fixtures and fakes on loopback.
- **A methodology change is recorded.** Changing the phases, the selection,
  the reduction or a table changes what an edition means: update
  [docs/methodology.md](docs/methodology.md) and say so in the CHANGELOG.
- **Nothing published may identify a server.**

## Licence

Code contributions are licensed under Apache-2.0; census data is CC-BY-4.0.
