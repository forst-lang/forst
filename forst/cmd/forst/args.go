package main

import (
	"flag"
	"fmt"
	"os"
)

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

func ParseArgs() ProgramArgs {
	if len(os.Args) < 2 {
		fmt.Println("Usage: forst <command> [flags] <filename>.ft")
		fmt.Println("\nCommands:")
		fmt.Println("  run    Compile and run the Forst program")
		return ProgramArgs{}
	}

	command := os.Args[1]
	if command != "run" {
		fmt.Printf("Unknown command: %s\n", command)
		fmt.Println("Supported commands: run")
		return ProgramArgs{}
	}

	// Create a new FlagSet for the run command
	runFlags := flag.NewFlagSet("run", flag.ExitOnError)
	debug := runFlags.Bool("debug", false, "Enable debug output")
	trace := runFlags.Bool("trace", false, "Enable trace output")
	watch := runFlags.Bool("watch", false, "Watch file for changes")
	output := runFlags.String("o", "", "Output file path")
	reportMemoryUsage := runFlags.Bool("report-memory-usage", false, "Report memory usage")
	reportPhases := runFlags.Bool("report-phases", false, "Report when phases start")

	if err := runFlags.Parse(os.Args[2:]); err != nil {
		return ProgramArgs{}
	}

	args := runFlags.Args()
	if len(args) < 1 {
		fmt.Println("Usage: forst run [-watch] [-o output] <filename>.ft")
		return ProgramArgs{}
	}

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
