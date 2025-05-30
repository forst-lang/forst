package main

import (
	"flag"
	"fmt"
	"os"
)

// ProgramArgs represents the arguments for the Forst compiler.
type ProgramArgs struct {
	command           string
	filePath          string
	outputPath        string
	debug             bool
	trace             bool
	watch             bool
	reportMemoryUsage bool
	reportPhases      bool
}

// ParseArgs parses the command line arguments and returns a ProgramArgs struct.
func ParseArgs() ProgramArgs {
	if len(os.Args) < 2 {
		printUsage()
		return ProgramArgs{}
	}

	command := os.Args[1]
	if command == "--help" || command == "-h" {
		printUsage()
		os.Exit(0)
	}

	if command != "run" && command != "build" {
		fmt.Printf("Unknown command: %s\n", command)
		fmt.Println("Supported commands: run, build")
		return ProgramArgs{}
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
		return ProgramArgs{}
	}

	if *help {
		flags.Usage()
		os.Exit(0)
	}

	args := flags.Args()
	if len(args) < 1 {
		fmt.Printf("Usage: forst %s [-o output] <filename>.ft\n", command)
		return ProgramArgs{}
	}

	// Fail if watch flag is provided with build command
	if command == "build" && *watch {
		fmt.Println("Error: -watch flag is not supported with build command")
		return ProgramArgs{}
	}

	// Require output path when using watch mode
	if *watch && *output == "" {
		fmt.Println("Error: -o flag required when using watch mode")
		return ProgramArgs{}
	}

	return ProgramArgs{
		command:           command,
		filePath:          args[0],
		outputPath:        *output,
		debug:             *debug,
		trace:             *trace,
		watch:             *watch,
		reportMemoryUsage: *reportMemoryUsage,
		reportPhases:      *reportPhases,
	}
}

func printUsage() {
	fmt.Println("Forst Compiler")
	fmt.Println("\nUsage: forst <command> [flags] <filename>.ft")
	fmt.Println("\nCommands:")
	fmt.Println("  run     Compile and run the Forst program")
	fmt.Println("  build   Compile the Forst program without running")
	fmt.Println("\nFlags:")
	fmt.Println("  -debug               Enable debug output")
	fmt.Println("  -trace               Enable trace output")
	fmt.Println("  -watch               Watch file for changes (run only)")
	fmt.Println("  -o <path>            Output file path")
	fmt.Println("  -report-memory-usage Report memory usage")
	fmt.Println("  -report-phases       Report when compilation phases start")
	fmt.Println("  -help                Show this help message")
}
