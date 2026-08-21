// Command carnet turns a trip directory — a GPX track, photos and a
// carnet.yaml — into publish-ready static content for an Astro site.
//
// This package is the only one allowed to call os.Exit or panic. Everything
// below it returns errors, which is what keeps the pipeline testable.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"carnet/internal/config"
)

// Exit codes of PRD §6.1. They are part of the CLI contract: a script piping
// carnet reads these, so they may never be renumbered.
const (
	exitOK      = 0
	exitRuntime = 1
	exitConfig  = 2
	exitPrivacy = 3
)

// No trailing blank line: main adds the newline it needs when printing.
const usage = `carnet — static generator for motorcycle trip logs

usage:
  carnet build <trip-dir> [options]   generate the static content
  carnet check <trip-dir> [options]   validate a trip directory, offline
  carnet version                      print the build version
`

func main() {
	// Ctrl-C and SIGTERM cancel ctx rather than killing the process outright.
	// Tile downloads hang on the network; without this they would leave a
	// half-written PNG behind.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	// Printing happens here, not in run: run stays a pure function of its
	// arguments, so a test reads the returned error instead of scraping stderr.
	err := run(ctx, os.Args[1:], os.Stdout, os.Stderr)

	// Released by hand rather than by defer: os.Exit below skips deferred calls,
	// and gocritic's exitAfterDefer refuses the pairing outright.
	stop()

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitCode(err))
	}
}

// printUsage discards the write error: nothing the caller could act on, and on
// the paths that print usage the returned error is the real message.
func printUsage(w io.Writer) {
	_, _ = fmt.Fprint(w, usage)
}

// run dispatches on the first argument and returns the error the exit code is
// derived from. Writers are injected rather than taken from os so tests can
// hand it buffers, and args excludes the program name.
//
// Both writers are already in the signature though no command writes yet: the
// report of check goes to stdout, its slog lines to stderr, and adding the
// parameters later would touch every call site.
func run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		printUsage(stderr)
		return errors.New("no command given")
	}

	switch args[0] {
	case "build":
		return errors.New("build: not implemented yet")
	case "check":
		return errors.New("check: not implemented yet")
	case "version":
		return errors.New("version: not implemented yet")
	default:
		printUsage(stderr)
		return fmt.Errorf("unknown command %q", args[0])
	}
}

// exitCode maps an error to one of the four codes above. It is the single
// place the mapping lives, so a new error type gets a code by being added here
// and nowhere else.
//
// The target is the leaf, *InvalidConfigError, never the ValidationErrors that
// usually holds it: the collection implements Unwrap() []error, so errors.As
// reaches through it, and a lone rejection built by check maps to 2 as well.
//
// nil maps to exitOK though main only calls on failure. The mapping is then
// total, and a test states it as one table.
//
// exitPrivacy has no case: internal/privacy owns no error type yet, and a case
// on a type that does not exist would not compile. S4 adds it.
func exitCode(err error) int {
	if err == nil {
		return exitOK
	}

	// Declared, never read. errors.As needs somewhere to write the match, and
	// the type of that destination is what it searches the chain for.
	var invalid *config.InvalidConfigError

	switch {
	case errors.As(err, &invalid):
		return exitConfig
	default:
		return exitRuntime
	}
}
