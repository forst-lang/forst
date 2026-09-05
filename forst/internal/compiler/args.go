package compiler

import (
	"flag"
	"os"
	"path/filepath"

	"forst/internal/devcompile"
	"forst/internal/programbuild"

	logrus "github.com/sirupsen/logrus"
)

var (
	osExit             = os.Exit
	filepathAbsForArgs = filepath.Abs
)

// Args represents the arguments for the Forst compiler.
type Args struct {
	// Command is the command to run (run, build, generate).
	Command string
	// FilePath is the path to the file to run.
	FilePath string
	// OutputPath is the path to the output file or directory.
	OutputPath string
	// LogLevel is the log level to use (debug, info, warn, error, trace).
	LogLevel string
	// Watch is true if the file should be watched for changes.
	Watch bool
	// ReportMemoryUsage is true if the memory usage should be reported.
	ReportMemoryUsage bool
	// ReportPhases is true if the phases should be reported.
	ReportPhases bool
	// ExportStructFields, when true, emits exported struct field names and json tags so encoding/json can marshal shapes (forst dev / ftconfig: compiler.exportStructFields).
	ExportStructFields bool
	// PackageRoot, if non-empty, enables merging all same-package .ft files that share one directory under this tree with the entry file (aligned with sidecar / discovery).
	PackageRoot string
	// RequireNoBridge, when true, fails the build if the program needs the script bridge runtime (opted-in JS imports).
	RequireNoBridge bool
	// ReloadProfile enables structured compile sub-phase timing logs for forst dev hot reload.
	ReloadProfile bool
	// DevStableSandbox reuses boundaryRoot/.forst/run/dev/ instead of a new temp dir each reload.
	DevStableSandbox bool
	// DevModTidyCache skips go mod tidy when go.mod content is unchanged across reloads.
	DevModTidyCache *SandboxModCache
	// DevSession holds incremental parse/module caches for dev reload (optional).
	DevSession *devcompile.Session
	// GoOS selects the target operating system for forst build (native program binary).
	GoOS string
	// GoARCH selects the target architecture for forst build (native program binary).
	GoARCH string
}

// ParseArgs parses os.Args for the run/build CLI path.
func ParseArgs(log *logrus.Logger) Args {
	return ParseArgsFrom(os.Args, log)
}

// ParseArgsFrom parses argv as the full argument vector (argv[0] is the program name).
// Used by cmd/forst after subcommands are handled, and by tests without mutating os.Args
// (avoids data races under -race with coverage teardown).
func ParseArgsFrom(argv []string, log *logrus.Logger) Args {
	if len(argv) < 2 {
		printUsage(log)
		return Args{}
	}

	command := argv[1]
	if command == "--help" || command == "-h" {
		printUsage(log)
		osExit(0)
	}

	if command == "--version" || command == "-v" {
		printVersion(log)
		osExit(0)
	}

	if command != "run" && command != "build" {
		log.Errorf("Unknown command: %s\n", command)
		log.Errorf("Supported commands: run, build")
		return Args{}
	}

	// Create a new FlagSet for the command
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(log.Writer())
	logLevel := flags.String("loglevel", "info", "Log level (debug, info, warn, error, trace)")
	watch := flags.Bool("watch", false, "Watch file for changes")
	output := flags.String("o", "", "Output file path")
	reportMemoryUsage := flags.Bool("report-memory-usage", false, "Report memory usage")
	reportPhases := flags.Bool("report-phases", false, "Report when phases start")
	exportStructFields := flags.Bool("export-struct-fields", false, "Emit exported struct fields with json tags (for encoding/json and TS-aligned wire shapes)")
	packageRoot := flags.String("root", "", "Root directory: merge same-package .ft files that share one directory with the entry file (optional)")
	requireNoBridge := flags.Bool("require-no-bridge", false, "Fail if the program requires the script bridge runtime (opted-in JavaScript imports)")
	goos := flags.String("goos", "", "Target GOOS for forst build (default: host)")
	goarch := flags.String("goarch", "", "Target GOARCH for forst build (default: host)")
	help := flags.Bool("help", false, "Show help message")

	if err := flags.Parse(argv[2:]); err != nil {
		return Args{}
	}

	if *help {
		flags.Usage()
		osExit(0)
	}

	args := flags.Args()
	if len(args) < 1 {
		log.Errorf("Usage: forst %s [-o output] <filename>.ft\n", command)
		return Args{}
	}

	// Fail if watch flag is provided with build command
	if command == "build" && *watch {
		log.Errorf("Error: -watch flag is not supported with build command")
		return Args{}
	}

	if command == "build" {
		if *output == "" {
			log.Errorf("Error: forst build requires -o <dir> (output directory for bin/<name> and manifest.json)")
			return Args{}
		}
		if err := programbuild.ValidateOutputPath(*output); err != nil {
			log.Errorf("%v", err)
			return Args{}
		}
	}

	if *goos != "" && command != "build" {
		log.Errorf("Error: --goos is only supported with forst build")
		return Args{}
	}

	if *goarch != "" && command != "build" {
		log.Errorf("Error: --goarch is only supported with forst build")
		return Args{}
	}

	if *packageRoot != "" && *watch {
		log.Errorf("Error: -root cannot be used with -watch")
		return Args{}
	}

	// Require output path when using watch mode
	if *watch && *output == "" {
		log.Errorf("Error: -o flag required when using watch mode")
		return Args{}
	}

	var pkgRoot string
	if *packageRoot != "" {
		abs, err := filepathAbsForArgs(*packageRoot)
		if err != nil {
			log.Errorf("invalid -root: %v", err)
			return Args{}
		}
		pkgRoot = abs
	}

	filePath := args[0]
	abs, err := filepathAbsForArgs(filePath)
	if err != nil {
		log.Errorf("invalid file path: %v", err)
		return Args{}
	}
	filePath = abs

	return Args{
		Command:            command,
		FilePath:           filePath,
		OutputPath:         *output,
		LogLevel:           *logLevel,
		Watch:              *watch,
		ReportMemoryUsage:  *reportMemoryUsage,
		ReportPhases:       *reportPhases,
		ExportStructFields: *exportStructFields,
		PackageRoot:        pkgRoot,
		RequireNoBridge:    *requireNoBridge,
		GoOS:               *goos,
		GoARCH:             *goarch,
	}
}

func printUsage(log *logrus.Logger) {
	log.Infof("Forst Compiler")
	log.Infof("\nUsage: forst <command> [flags] <filename>.ft")
	log.Infof("\nCommands:")
	log.Infof("  dev     Start the Forst development server")
	log.Infof("  lsp     Start the Forst LSP server")
	log.Infof("  run     Compile and run a Forst program")
	log.Infof("  build   Build a native program binary and manifest under -o <dir>")
	log.Infof("  generate Generate TypeScript client code (optional Go sources with --go)")
	log.Infof("\nFlags:")
	log.Infof("  -loglevel <level>       Log level (debug, info, warn, error, trace)")
	log.Infof("  -watch                  Watch file for changes (run only)")
	log.Infof("  -o <path>               Output path (build: directory; run watch: file)")
	log.Infof("  --goos <os>             Target GOOS for forst build (default: host)")
	log.Infof("  --goarch <arch>         Target GOARCH for forst build (default: host)")
	log.Infof("  -report-memory-usage    Report memory usage")
	log.Infof("  -report-phases          Report when phases start")
	log.Infof("  -export-struct-fields   Emit exported struct fields with json tags for JSON marshaling")
	log.Infof("  -root <dir>             Merge same-package .ft files under dir with the entry file")
	log.Infof("  -require-no-bridge      Fail if the program requires the script bridge runtime")
	log.Infof("  -help                   Show this help message")
	log.Infof("  -version                Show version information")
}

func printVersion(log *logrus.Logger) {
	log.Infof("Forst Compiler v%s", version)
	log.Infof("Commit: %s", commit)
	log.Infof("Date: %s", date)
}
