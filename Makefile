# SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
# SPDX-License-Identifier: Apache-2.0

.PHONY: all build test test-race coverage coverage-json vet lint format spdx-check reuse readme-check \
        name-guard versions manual passmcp census-list census help

# The version is the newest release heading in CHANGELOG.md, and nowhere else.
VERSION ?= $(shell grep -Eo '^.. \[[0-9]+\.[0-9]+\.[0-9]+\]' CHANGELOG.md | head -1 | tr -dc '0-9.')

# The passmcp release a census runs: a tagged release that exists, never
# newer than this repository's own version. Every document that names it
# must agree (scripts/verify-release-versions.sh).
PASSMCP_VERSION := v0.0.2
PASSMCP ?= $(CURDIR)/build/bin/passmcp
REGISTRY ?= https://registry.modelcontextprotocol.io
EDITION ?=

# Every gate CI runs that needs no network, cheap ones first.
all: format vet lint spdx-check readme-check name-guard versions test

build:
	CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.Version=$(VERSION)" \
	  -o build/passmcp-census ./cmd/passmcp-census

test:
	go test ./... -cover

test-race:
	go test -race -shuffle=on -count=1 ./...

# The gate is 85% statement coverage in every package; ci.yml applies it.
coverage:
	go test -count=1 -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -1

# The shields.io endpoint document behind the README's coverage badge. The
# Manual workflow publishes it with GitHub Pages as coverage.json.
coverage-json: coverage
	@mkdir -p build
	go run ./scripts/coveragebadge -profile coverage.out > build/coverage.json
	@cat build/coverage.json

vet:
	go vet ./...

lint:
	golangci-lint run ./...

format:
	gofmt -l -w .

spdx-check:
	go run ./scripts/spdx_sweep.go

reuse:
	reuse lint

readme-check:
	scripts/readme-check.sh

name-guard:
	scripts/name-guard.sh

versions:
	scripts/verify-release-versions.sh "v$(VERSION)"

# The rendered manual, strictly. Needs the hash-locked requirements:
# pip install --require-hashes -r docs/requirements.txt
manual:
	mkdocs build --strict --site-dir public

# The pinned passmcp release, into build/bin.
passmcp:
	GOBIN="$(CURDIR)/build/bin" go install "satellion.com/passmcp/cmd/passmcp@$(PASSMCP_VERSION)"

# The registry snapshot and what a run would check. Reads the registry;
# contacts no MCP server.
census-list: build
	@test -n "$(EDITION)" || { echo "census-list: set EDITION=YYYY-MM" >&2; exit 2; }
	build/passmcp-census list --edition "$(EDITION)" --registry "$(REGISTRY)"

# A census edition, end to end: the pinned passmcp checks every listed
# remote endpoint read-only and without credentials into build/census/,
# then the aggregate dataset is written to data/$(EDITION)/. This contacts
# every listed server: it is the maintainer's decision, never CI's
# (docs/adr/0003-editions-run-by-hand.md).
census: build passmcp
	@test -n "$(EDITION)" || { echo "census: set EDITION=YYYY-MM" >&2; exit 2; }
	build/passmcp-census run --edition "$(EDITION)" --registry "$(REGISTRY)" --passmcp "$(PASSMCP)"
	build/passmcp-census aggregate --edition "$(EDITION)"

help:
	@printf '%s\n' "targets: all build test test-race coverage coverage-json vet lint format spdx-check reuse" \
	  "         readme-check name-guard versions manual passmcp census-list census (EDITION=YYYY-MM)"
