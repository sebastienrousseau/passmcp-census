# SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
# SPDX-License-Identifier: Apache-2.0
#
# The README demo: a whole census edition on loopback. A fake registry
# (.github/demo/registry, served by python3's http.server) lists three of
# passmcp's example servers: the baseline, any-origin and wrong-id. `run`
# checks them with the pinned passmcp, `aggregate` writes the dataset, and
# the per-phase table shows the two protocol failures. Rendered by
# `make demo`, which runs this in build/demo/work with its own copy of
# data/exclusions.txt, so the edition it writes never reaches data/.
Output .github/demo.gif
Require passmcp
Require passmcp-census
Require servers
Require python3
Set Shell "bash"
Set FontSize 13
Set Width 1200
Set Height 600
Set Padding 20
Set Theme "Catppuccin Mocha"
Set TypingSpeed 35ms

Hide
# Everything stops itself after 90s: a hidden kill would still print a line.
Type "PS1='$ '; cd build/demo/work" Enter
Type "timeout 90 python3 -m http.server 17820 --bind 127.0.0.1 -d ../../../.github/demo/registry >/dev/null 2>&1 &" Enter
Type "timeout 90 servers -flaw baseline -addr 127.0.0.1:17821 >/dev/null 2>&1 &" Enter
Type "timeout 90 servers -flaw any-origin -addr 127.0.0.1:17822 >/dev/null 2>&1 &" Enter
Type "timeout 90 servers -flaw wrong-id -addr 127.0.0.1:17823 >/dev/null 2>&1 &" Enter
# Let the set-up finish and clear it before recording starts.
Sleep 2s
Type "clear" Enter
Sleep 500ms
Show

Type "passmcp-census run --edition 2026-09 --registry http://127.0.0.1:17820 --host-interval 1s > /dev/null" Sleep 400ms Enter
Sleep 5s
Type "passmcp-census aggregate --edition 2026-09" Sleep 400ms Enter
Sleep 2s
Type "cat data/2026-09/by-phase.csv" Sleep 400ms Enter
Sleep 4s
