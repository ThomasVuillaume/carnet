package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"carnet/internal/config"
)

// The table states the CLI contract of PRD §6.1: which arguments produce which
// exit code, and which of them answer with the usage.
//
// run never prints the error — main does — so wantErr is matched against the
// returned value while the buffers hold only usage and log lines. That split is
// what lets stderr be asserted as empty on the paths that must stay quiet.
//
// Substrings rather than whole outputs: the usage text and the wording of the
// errors are meant to be edited, the exit codes and the routing are not.
func TestRun(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantErr    string   // substring of the returned error; "" means nil
		wantUsage  bool     // the usage reached stderr
		wantStdout string   // substring; "" means stdout stayed empty
		wantStderr []string // substrings beyond the usage
	}{
		{
			name:      "no arguments",
			args:      nil,
			wantCode:  exitRuntime,
			wantErr:   "no command given",
			wantUsage: true,
		},
		{
			name:      "help command",
			args:      []string{"help"},
			wantCode:  exitOK,
			wantUsage: true,
		},
		// The four spellings flag would have recognised on a subcommand, listed
		// by hand at the top level. All must reach the same arm.
		{
			name:      "help flag, one dash",
			args:      []string{"-h"},
			wantCode:  exitOK,
			wantUsage: true,
		},
		{
			name:      "help flag, two dashes",
			args:      []string{"--h"},
			wantCode:  exitOK,
			wantUsage: true,
		},
		{
			name:      "long help flag, one dash",
			args:      []string{"-help"},
			wantCode:  exitOK,
			wantUsage: true,
		},
		{
			name:      "long help flag, two dashes",
			args:      []string{"--help"},
			wantCode:  exitOK,
			wantUsage: true,
		},
		{
			name:       "version goes to stdout",
			args:       []string{"version"},
			wantCode:   exitOK,
			wantStdout: binaryName,
		},
		{
			name:      "unknown command",
			args:      []string{"nope"},
			wantCode:  exitRuntime,
			wantErr:   `unknown command "nope"`,
			wantUsage: true,
		},
		{
			name:      "build without a trip directory",
			args:      []string{"build"},
			wantCode:  exitRuntime,
			wantErr:   "build: missing trip directory",
			wantUsage: true,
		},
		{
			name:      "check without a trip directory",
			args:      []string{"check"},
			wantCode:  exitRuntime,
			wantErr:   "check: missing trip directory",
			wantUsage: true,
		},
		{
			name:      "unknown flag",
			args:      []string{"build", "--nope", "./trip"},
			wantCode:  exitRuntime,
			wantErr:   "not defined",
			wantUsage: true,
		},
		// flag stops at the first positional, so a trailing option is counted
		// as a second directory rather than silently ignored.
		{
			name:      "flag after the trip directory",
			args:      []string{"build", "./trip", "--verbose"},
			wantCode:  exitRuntime,
			wantErr:   "expected one trip directory",
			wantUsage: true,
		},
		{
			name:      "help flag on a subcommand",
			args:      []string{"build", "-h"},
			wantCode:  exitOK,
			wantUsage: true,
		},
		{
			name:     "build reaches the pipeline and stays quiet",
			args:     []string{"build", "./trip"},
			wantCode: exitRuntime,
			wantErr:  "build: not implemented yet",
		},
		{
			name:     "check reaches the pipeline and stays quiet",
			args:     []string{"check", "./trip"},
			wantCode: exitRuntime,
			wantErr:  "check: not implemented yet",
		},
		// The whole logging chain in one row: the threshold lets DEBUG through,
		// AddSource fires, the path is shortened, and With bound the command.
		{
			name:     "verbose opens the debug level",
			args:     []string{"build", "--verbose", "./trip"},
			wantCode: exitRuntime,
			wantErr:  "not implemented yet",
			wantStderr: []string{
				"level=DEBUG",
				"source=carnet/main.go:",
				`msg="options parsed"`,
				"command=build",
				"trip_dir=./trip",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var stdout, stderr bytes.Buffer

			err := run(t.Context(), tc.args, &stdout, &stderr)

			if got := exitCode(err); got != tc.wantCode {
				t.Errorf("exit code = %d, want %d (err = %v)", got, tc.wantCode, err)
			}

			switch {
			case tc.wantErr == "" && err != nil:
				t.Errorf("unexpected error: %v", err)
			case tc.wantErr != "" && err == nil:
				t.Errorf("no error, want one containing %q", tc.wantErr)
			case tc.wantErr != "" && !strings.Contains(err.Error(), tc.wantErr):
				t.Errorf("error = %q, want it to contain %q", err, tc.wantErr)
			}

			if gotUsage := strings.Contains(stderr.String(), usage); gotUsage != tc.wantUsage {
				t.Errorf("usage printed = %v, want %v", gotUsage, tc.wantUsage)
			}

			if tc.wantStdout == "" && stdout.Len() > 0 {
				t.Errorf("stdout = %q, want empty", stdout.String())
			}
			if tc.wantStdout != "" && !strings.Contains(stdout.String(), tc.wantStdout) {
				t.Errorf("stdout = %q, want it to contain %q", stdout.String(), tc.wantStdout)
			}

			// A row that wants neither usage nor log lines wants silence.
			if !tc.wantUsage && len(tc.wantStderr) == 0 && stderr.Len() > 0 {
				t.Errorf("stderr = %q, want empty", stderr.String())
			}
			for _, want := range tc.wantStderr {
				if !strings.Contains(stderr.String(), want) {
					t.Errorf("stderr = %q, want it to contain %q", stderr.String(), want)
				}
			}
		})
	}
}

// The table walks every depth at which an InvalidConfigError can sit, because
// errors.As has to reach it through all of them: bare, wrapped by fmt, held by
// ValidationErrors through its plural Unwrap, and both at once. Anything else
// is a runtime error.
//
// exitPrivacy is absent: internal/privacy owns no error type yet.
func TestExitCode(t *testing.T) {
	t.Parallel()

	invalid := &config.InvalidConfigError{Key: "map.max_zoom", Value: 42, Reason: "out of range"}
	collection := config.ValidationErrors{invalid}.Err()

	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "no error", err: nil, want: exitOK},
		{name: "plain error", err: errors.New("boom"), want: exitRuntime},
		{name: "invalid config, bare", err: invalid, want: exitConfig},
		{name: "invalid config, wrapped", err: fmt.Errorf("loading: %w", invalid), want: exitConfig},
		{name: "invalid config, in the collection", err: collection, want: exitConfig},
		{name: "collection, wrapped", err: fmt.Errorf("carnet.yaml: %w", collection), want: exitConfig},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := exitCode(tc.err); got != tc.want {
				t.Errorf("exitCode(%v) = %d, want %d", tc.err, got, tc.want)
			}
		})
	}
}

// The table discriminates what flag does with argument order and spelling —
// the behaviour that is not obvious and that the usage text depends on.
//
// On every rejection the returned options must be the zero value, never the
// half-filled struct: a caller that skipped the error check would otherwise run
// the pipeline on an empty trip directory.
func TestParseOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		args     []string
		want     options
		wantErr  string // substring; "" means nil
		wantHelp bool   // flag.ErrHelp, which is not a failure
	}{
		{
			name: "flag then directory",
			args: []string{"--verbose", "./trip"},
			want: options{verbose: true, tripDir: "./trip"},
		},
		// flag treats one dash and two as the same thing.
		{
			name: "one dash",
			args: []string{"-verbose", "./trip"},
			want: options{verbose: true, tripDir: "./trip"},
		},
		{
			name: "directory alone",
			args: []string{"./trip"},
			want: options{tripDir: "./trip"},
		},
		// A bool flag takes its value inline or not at all: "--verbose false"
		// would read false as the directory.
		{
			name: "bool set explicitly",
			args: []string{"--verbose=false", "./trip"},
			want: options{tripDir: "./trip"},
		},
		// "--" ends flag parsing, so what follows is positional whatever it
		// looks like.
		{
			name: "terminator makes a flag positional",
			args: []string{"--", "--verbose"},
			want: options{tripDir: "--verbose"},
		},
		{
			name:    "no arguments",
			args:    nil,
			wantErr: "build: missing trip directory",
		},
		{
			name:    "flag after the directory",
			args:    []string{"./trip", "--verbose"},
			wantErr: "build: expected one trip directory, got 2",
		},
		{
			name:    "unknown flag",
			args:    []string{"--nope", "./trip"},
			wantErr: "build: flag provided but not defined",
		},
		{
			name:     "help requested",
			args:     []string{"-h"},
			wantHelp: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseOptions("build", tc.args)

			if tc.wantHelp {
				if !errors.Is(err, flag.ErrHelp) {
					t.Fatalf("error = %v, want flag.ErrHelp", err)
				}

				return
			}

			switch {
			case tc.wantErr == "" && err != nil:
				t.Fatalf("unexpected error: %v", err)
			case tc.wantErr != "" && err == nil:
				t.Fatalf("no error, want one containing %q", tc.wantErr)
			case tc.wantErr != "" && !strings.Contains(err.Error(), tc.wantErr):
				t.Fatalf("error = %q, want it to contain %q", err, tc.wantErr)
			}

			if got != tc.want {
				t.Errorf("options = %+v, want %+v", got, tc.want)
			}
		})
	}
}

// Two rows, and they freeze a decision rather than a computation: Info is the
// floor without --verbose, so the lines proving the privacy purge ran stay
// visible by default.
func TestLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		verbose bool
		want    slog.Level
	}{
		{name: "quiet", verbose: false, want: slog.LevelInfo},
		{name: "verbose", verbose: true, want: slog.LevelDebug},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := level(tc.verbose); got != tc.want {
				t.Errorf("level(%v) = %v, want %v", tc.verbose, got, tc.want)
			}
		})
	}
}

// The table separates the two attributes the handler adds by itself from
// everything a caller writes. Dropping the clock and the build machine's paths
// is what makes two runs of the same trip produce identical stderr; touching a
// caller's attribute that happens to share a key would be a bug.
func TestReplaceAttr(t *testing.T) {
	t.Parallel()

	stamp := time.Date(2026, time.August, 22, 14, 3, 11, 0, time.UTC)

	// One pointer, used on both sides of the pass-through rows. slog.Value
	// compares an Any by identity, so two Sources with equal fields are not
	// equal — and identity is what those rows assert anyway.
	grouped := &slog.Source{File: "/home/someone/carnet/internal/gpx/parse.go", Line: 97}

	tests := []struct {
		name   string
		groups []string
		attr   slog.Attr
		want   slog.Attr
	}{
		{
			name: "timestamp dropped",
			attr: slog.Time(slog.TimeKey, stamp),
			want: slog.Attr{},
		},
		{
			name: "source cut to package and file",
			attr: slog.Any(slog.SourceKey, &slog.Source{
				File: "/home/someone/GitRepositories/carnet/internal/gpx/parse.go",
				Line: 97,
			}),
			want: slog.String(slog.SourceKey, "gpx/parse.go:97"),
		},
		{
			name: "source already one segment deep",
			attr: slog.Any(slog.SourceKey, &slog.Source{File: "/carnet/main.go", Line: 1}),
			want: slog.String(slog.SourceKey, "carnet/main.go:1"),
		},
		// The key is only the handler's when the value is too.
		{
			name: "source key holding a string",
			attr: slog.String(slog.SourceKey, "hand written"),
			want: slog.String(slog.SourceKey, "hand written"),
		},
		{
			name: "source key holding a typed nil",
			attr: slog.Any(slog.SourceKey, (*slog.Source)(nil)),
			want: slog.Any(slog.SourceKey, (*slog.Source)(nil)),
		},
		// Inside a group the keys belong to whoever built the group.
		{
			name:   "timestamp inside a group survives",
			groups: []string{"photo"},
			attr:   slog.Time(slog.TimeKey, stamp),
			want:   slog.Time(slog.TimeKey, stamp),
		},
		{
			name:   "source inside a group survives",
			groups: []string{"photo"},
			attr:   slog.Any(slog.SourceKey, grouped),
			want:   slog.Any(slog.SourceKey, grouped),
		},
		{
			name: "an ordinary attribute passes through",
			attr: slog.Int("zoom", 12),
			want: slog.Int("zoom", 12),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := replaceAttr(tc.groups, tc.attr)

			// slog.Attr holds a Value, which compares through Equal rather
			// than ==: a Value can carry a pointer or a time.Time.
			if !got.Equal(tc.want) {
				t.Errorf("replaceAttr(%v, %v) = %v, want %v", tc.groups, tc.attr, got, tc.want)
			}
		})
	}
}

// The guard for the choice made in silencing the FlagSet: usage is now the only
// help text a user ever sees, so a flag declared and left undocumented would be
// invisible. Walking the FlagSet rather than a second list keeps the check
// honest when a flag is added.
func TestUsageDocumentsEveryFlag(t *testing.T) {
	t.Parallel()

	var opts options

	newFlagSet("build", &opts).VisitAll(func(f *flag.Flag) {
		if !strings.Contains(usage, "--"+f.Name) {
			t.Errorf("flag --%s is declared but absent from usage", f.Name)
		}
	})
}
