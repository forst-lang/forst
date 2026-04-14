package compiler

import (
	"flag"
	"os"
	"path/filepath"

	logrus "github.com/sirupsen/logrus"
)

// Import version variables from main package
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

// Args represents the arguments for the Forst compiler.
type Args struct {
	Command           string
	FilePath          string
	OutputPath        string
	LogLevel          string
	Watch             bool
	ReportMemoryUsage bool
	ReportPhases      bool
	// ExportStructFields, when true, emits exported struct field names and json tags so encoding/json can marshal shapes (forst dev / ftconfig: compiler.exportStructFields).
	ExportStructFields bool
	// PackageRoot, if non-empty, enables merging all same-package .ft files under this directory with the entry file (aligned with sidecar / discovery).
	PackageRoot string
}

// ParseArgs parses the command line arguments and returns a ProgramArgs struct.
func ParseArgs(log *logrus.Logger) Args {
	if len(os.Args) < 2 {
		printUsage(log)
		return Args{}
	}

	command := os.Args[1]
	if command == "--help" || command == "-h" {
		printUsage(log)
		os.Exit(0)
	}

	if command == "--version" || command == "-v" {
		printVersion(log)
		os.Exit(0)
	}

	if command != "run" && command != "build" {
		log.Errorf("Unknown command: %s\n", command)
		log.Errorf("Supported commands: run, build")
		return Args{}
	}

	// Create a new FlagSet for the command
	flags := flag.NewFlagSet(command, flag.ExitOnError)
	logLevel := flags.String("loglevel", "info", "Log level (debug, info, warn, error, trace)")
	watch := flags.Bool("watch", false, "Watch file for changes")
	output := flags.String("o", "", "Output file path")
	reportMemoryUsage := flags.Bool("report-memory-usage", false, "Report memory usage")
	reportPhases := flags.Bool("report-phases", false, "Report when phases start")
	exportStructFields := flags.Bool("export-struct-fields", false, "Emit exported struct fields with json tags (for encoding/json and TS-aligned wire shapes)")
	packageRoot := flags.String("root", "", "Root directory: merge all .ft files under it that share the entry file's package (optional)")
	help := flags.Bool("help", false, "Show help message")

	if err := flags.Parse(os.Args[2:]); err != nil {
		return Args{}
	}

	if *help {
		flags.Usage()
		os.Exit(0)
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
		abs, err := filepath.Abs(*packageRoot)
		if err != nil {
			log.Errorf("invalid -root: %v", err)
			return Args{}
		}
		pkgRoot = abs
	}

	return Args{
		Command:            command,
		FilePath:           args[0],
		OutputPath:         *output,
		LogLevel:           *logLevel,
		Watch:              *watch,
		ReportMemoryUsage:  *reportMemoryUsage,
		ReportPhases:       *reportPhases,
		ExportStructFields: *exportStructFields,
		PackageRoot:        pkgRoot,
	}
}

func printUsage(log *logrus.Logger) {
	log.Infof("Forst Compiler")
	log.Infof("\nUsage: forst <command> [flags] <filename>.ft")
	log.Infof("\nCommands:")
	log.Infof("  dev     Start the Forst development server")
	log.Infof("  lsp     Start the Forst LSP server")
	log.Infof("  run     Compile and run a Forst program")
	log.Infof("  build   Compile a Forst program without running")
	log.Infof("  generate Generate TypeScript client code")
	log.Infof("\nFlags:")
	log.Infof("  -loglevel <level>       Log level (debug, info, warn, error, trace)")
	log.Infof("  -watch                  Watch file for changes (run only)")
	log.Infof("  -o <path>               Output file path")
	log.Infof("  -report-memory-usage    Report memory usage")
	log.Infof("  -report-phases          Report when phases start")
	log.Infof("  -export-struct-fields   Emit exported struct fields with json tags for JSON marshaling")
	log.Infof("  -root <dir>             Merge same-package .ft files under dir with the entry file")
	log.Infof("  -help                   Show this help message")
	log.Infof("  -version                Show version information")
}

func printVersion(log *logrus.Logger) {
	log.Infof("Forst Compiler v%s", version)
	log.Infof("Commit: %s", commit)
	log.Infof("Date: %s", date)
}
