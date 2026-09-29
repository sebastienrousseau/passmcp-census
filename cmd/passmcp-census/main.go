// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// Command passmcp-census runs and aggregates the passmcp reliability census.
//
//	passmcp-census list --edition 2026-09
//	passmcp-census run --edition 2026-09 --passmcp "$(command -v passmcp)"
//	passmcp-census aggregate --edition 2026-09
//	passmcp-census version
//
// run lists the MCP Registry's remote servers and checks each with passmcp,
// read-only and without credentials, into a private working directory.
// aggregate turns that directory into the published dataset: counts and
// rates, no server identified. Results go to stdout as JSON; progress and
// every diagnostic go to stderr.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
)

// Version is set at build time with -ldflags "-X main.Version=…".
var Version = "dev"

const usage = `usage: passmcp-census <command> [flags]

commands:
  list       list the registry and count what a run would check, contacting no server
  run        list the registry and check every remote endpoint with passmcp
  aggregate  turn a run's working directory into the published dataset
  version    print the version

"passmcp-census <command> -h" lists a command's flags.
`

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := run(ctx, os.Args[1:], os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}

// run is the command without the process around it.
func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprint(stderr, usage)
		return 2
	}
	switch args[0] {
	case "list":
		return cmdList(ctx, args[1:], stdout, stderr)
	case "run":
		return cmdRun(ctx, args[1:], stdout, stderr)
	case "aggregate":
		return cmdAggregate(args[1:], stdout, stderr)
	case "version", "--version":
		_, _ = fmt.Fprintln(stdout, "passmcp-census", Version)
		return 0
	case "help", "-h", "--help":
		_, _ = fmt.Fprint(stdout, usage)
		return 0
	default:
		_, _ = fmt.Fprintf(stderr, "passmcp-census: unknown command %q\n\n%s", args[0], usage)
		return 2
	}
}

func fail(stderr io.Writer, err error) int {
	_, _ = fmt.Fprintln(stderr, "passmcp-census:", err)
	return 1
}
