#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
# SPDX-License-Identifier: Apache-2.0
#
# Fail unless every place that names the version names the same one: the
# CHANGELOG heading, the release highlights, the README's ecosystem
# sentence and CITATION.cff. The family moves in lockstep, so all of them
# are the same version.
#
# The passmcp release the Makefile pins for a census (PASSMCP_VERSION) is
# a sibling's release, which must already exist: it may trail this
# version, never lead it, and every go install line and document that
# names it must name the same release.
#
#   scripts/verify-release-versions.sh v0.0.2
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."
tag="${1:-${GITHUB_REF_NAME:-}}"
[ -n "${tag}" ] || { echo "usage: $0 vX.Y.Z" >&2; exit 2; }
[[ "${tag}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo "not a release tag: ${tag}" >&2; exit 2; }
ver="${tag#v}"
fail=0
bad() { echo "$*" >&2; fail=1; }

grep -Eq "^## \[${ver}\]" CHANGELOG.md || bad "CHANGELOG.md has no '## [${ver}]' heading"

notes="docs/releases/v${ver}.md"
if [ ! -f "${notes}" ]; then
  bad "${notes} is missing"
elif ! grep -Fq '## Highlights ⭐️' "${notes}"; then
  bad "${notes} has no '## Highlights ⭐️' section"
fi

grep -Fq "Every component is released at **${ver}**" README.md ||
  bad "README.md's ecosystem section does not state ${ver}"

grep -Eq "^version: \"?${ver}\"?\$" CITATION.cff || bad "CITATION.cff does not say version ${ver}"

pin=$(sed -n 's/^PASSMCP_VERSION := \(v[0-9][0-9.]*\)$/\1/p' Makefile)
if ! [[ "${pin}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  bad "Makefile does not pin passmcp to a release (PASSMCP_VERSION := vX.Y.Z)"
elif [ "$(printf '%s\n%s\n' "${pin#v}" "${ver}" | sort -V | tail -1)" != "${ver}" ]; then
  bad "Makefile pins passmcp ${pin}, newer than this release ${ver}"
fi

docs=(README.md DEVELOPMENT.md docs/*.md docs/adr/*.md)
check_installs() { # <module suffix> <want>: every go install line of it names <want>
  local lines
  lines=$(grep -Eoh "satellion\.com/$1/cmd/$1@[^ \`)\"]+" "${docs[@]}" || true)
  if grep -v "@$2\$" <<<"${lines}" | grep .; then
    bad "the docs pin a go install of $1 other than $2"
  fi
}
grep -Eq "satellion\.com/passmcp/cmd/passmcp@" README.md || bad "README.md names no passmcp go install line"
check_installs passmcp "${pin}"
check_installs passmcp-census "v${ver}"

grep -Fq "the release the Makefile pins, ${pin} " README.md ||
  bad "README.md's requirements do not name the pinned passmcp ${pin}"
grep -Fq "(\`PASSMCP_VERSION\`, ${pin})" docs/methodology.md ||
  bad "docs/methodology.md does not name the pinned passmcp ${pin}"

[ "${fail}" -eq 0 ] && echo "release versions agree on ${ver}"
exit "${fail}"
