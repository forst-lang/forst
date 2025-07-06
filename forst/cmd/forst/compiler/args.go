package compiler

import (
	"flag"
	"os"

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
	Debug             bool
	Trace             bool
	Watch             bool
	ReportMemoryUsage bool
	ReportPhases      bool
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
	debug := flags.Bool("debug", false, "Enable debug output")
	trace := flags.Bool("trace", false, "Enable trace output")
	watch := flags.Bool("watch", false, "Watch file for changes")
	output := flags.String("o", "", "Output file path")
	reportMemoryUsage := flags.Bool("report-memory-usage", false, "Report memory usage")
	reportPhases := flags.Bool("report-phases", false, "Report when phases start")
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

	// Require output path when using watch mode
	if *watch && *output == "" {
		log.Errorf("Error: -o flag required when using watch mode")
		return Args{}
	}

	return Args{
		Command:           command,
		FilePath:          args[0],
		OutputPath:        *output,
		Debug:             *debug,
		Trace:             *trace,
		Watch:             *watch,
		ReportMemoryUsage: *reportMemoryUsage,
		ReportPhases:      *reportPhases,
	}
}

func printUsage(log *logrus.Logger) {
	log.Infof("Forst Compiler")
	log.Infof("\nUsage: forst <command> [flags] <filename>.ft")
	log.Infof("\nCommands:")
	log.Infof("  run     Compile and run the Forst program")
	log.Infof("  build   Compile the Forst program without running")
	log.Infof("\nFlags:")
	log.Infof("  -debug               Enable debug output")
	log.Infof("  -trace               Enable trace output")
	log.Infof("  -watch               Watch file for changes (run only)")
	log.Infof("  -o <path>            Output file path")
	log.Infof("  -report-memory-usage Report memory usage")
	log.Infof("  -report-phases       Report when compilation phases start")
	log.Infof("  -help                Show this help message")
	log.Infof("  -version             Show version information")
}

func printVersion(log *logrus.Logger) {
	log.Infof("Forst Compiler v%s", version)
	log.Infof("Commit: %s", commit)
	log.Infof("Date: %s", date)
}
