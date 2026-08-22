// Command carnet turns a trip directory — a GPX track, photos and a
// carnet.yaml — into publish-ready static content for an Astro site.
//
// This package is the only one allowed to call os.Exit or panic. Everything
// below it returns errors, which is what keeps the pipeline testable.
package main

import (
	"context"
	"errors"
	"flag"
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

options:
  --verbose                           log at debug level
  -h, --help                          print this help
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
// stdout carries command output — version prints there, the report of check
// will follow. stderr carries usage and the slog lines.
func run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		printUsage(stderr)
		return errors.New("no command given")
	}

	switch args[0] {
	// One arm for both: build and check take the same trip directory and the
	// same options. They diverge once the pipeline exists, not before.
	case "build", "check":
		opts, err := parseOptions(args[0], args[1:])

		switch {
		// flag answers -h and -help without being told to, and reports it as
		// this sentinel. Help was asked for and given, so the command
		// succeeded: exit 0, no error line. Printing our own usage rather than
		// the FlagSet's keeps one help text whatever route reaches it.
		case errors.Is(err, flag.ErrHelp):
			printUsage(stderr)
			return nil

		// Every other rejection is a usage mistake — unknown flag, missing or
		// doubled trip directory — so all of them get the same answer. Printing
		// here and not inside parseOptions leaves that one a pure function, and
		// keeps every write to stderr on this level.
		case err != nil:
			printUsage(stderr)
			return err
		}

		// The root of the logger tree. With binds the command once; every line
		// written from here down carries it without naming it again, and the
		// pipeline stages below add their own key — stage=tiles, stage=privacy
		// — to the same record. Never "stage" here: With appends rather than
		// overwrites, so a key reused down the tree prints twice.
		log := newLogger(stderr, opts.verbose).With("command", args[0])
		log.Debug("options parsed", "trip_dir", opts.tripDir)

		return fmt.Errorf("%s: not implemented yet", args[0])
	// No FlagSet parses this level — the switch reads a bare string — so the
	// spellings flag would have recognised are listed by hand. All four: flag
	// accepts one dash or two indifferently, and "carnet --h" printing usage
	// while "carnet build --h" errored would be the kind of seam users report.
	case "help", "-h", "--h", "-help", "--help":
		printUsage(stderr)
		return nil
	case "version":
		return printVersion(stdout)
	default:
		printUsage(stderr)
		return fmt.Errorf("unknown command %q", args[0])
	}
}

// options is what build and check accept, after parsing.
type options struct {
	verbose bool
	tripDir string
}

// newFlagSet declares the flags build and check share, bound to opts.
//
// Split out of parseOptions so a test can walk the declared flags and assert
// that usage documents every one of them. Silencing the FlagSet made that
// constant the only help text users see, and nothing else keeps the two in
// step.
//
// Silenced because Parse prints its message and returns it: any wired-up
// writer reports every flag error twice, once by the FlagSet and once by main.
// The returned error is the single message, and run prints the usage.
func newFlagSet(name string, opts *options) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.BoolVar(&opts.verbose, "verbose", false, "log at debug level")

	return fs
}

// parseOptions parses the flags and the positional trip directory of one
// subcommand. name is that subcommand, args excludes it.
//
// Package flag has no notion of subcommands. A plain flag.Parse reads os.Args
// whole, stops at "build" — the first argument not starting with a dash — and
// never sees the flags behind it. Splitting the arguments at the subcommand
// and parsing the remainder is the gap this fills, the same gap cobra and
// urfave/cli exist to close. go.mod rules both out.
//
// The FlagSet is built here rather than taken from flag.CommandLine: that one
// is a mutable global, and two tests parsing different arguments would share
// it. ContinueOnError makes Parse return its error instead of calling
// os.Exit — a call only main is allowed to make.
//
// flag stops at the first non-flag argument, so "carnet build ./trip
// --verbose" leaves --verbose positional. The count check below is what turns
// that into a message rather than a silently ignored flag.
func parseOptions(name string, args []string) (options, error) {
	var opts options

	fs := newFlagSet(name, &opts)

	if err := fs.Parse(args); err != nil {
		return options{}, fmt.Errorf("%s: %w", name, err)
	}

	// PRD §6.1: the trip directory is positional and mandatory. Parse hands
	// back the leftover arguments and never counts them, so the arity is
	// checked here or nowhere.
	switch fs.NArg() {
	case 0:
		return options{}, fmt.Errorf("%s: missing trip directory", name)
	case 1:
		opts.tripDir = fs.Arg(0)
	default:
		return options{}, fmt.Errorf("%s: expected one trip directory, got %d — flags must precede it", name, fs.NArg())
	}

	return opts, nil
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
