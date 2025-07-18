// Package main is the main package for the forst compiler.
package main

import (
	"flag"
	"fmt"
	"forst/cmd/forst/compiler"
	"forst/cmd/forst/lsp"
	"os"
	"path/filepath"

	logrus "github.com/sirupsen/logrus"
)

// Version information injected by Release Please
var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

func printVersionInfo() {
	fmt.Printf("forst %s %s %s\n", Version, Commit, Date)
}

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "version" || os.Args[1] == "--version" || os.Args[1] == "-v") {
		printVersionInfo()
		os.Exit(0)
	}

	// Create logger with appropriate level based on build type
	var log *logrus.Logger
	if Version == "dev" {
		log = logrus.New()
		log.SetLevel(logrus.DebugLevel)
	} else {
		log = logrus.New()
		log.SetLevel(logrus.InfoLevel)
	}

	// Check if we should start dev server
	if len(os.Args) > 1 && os.Args[1] == "dev" {
		// Parse flags for dev server
		devFlags := flag.NewFlagSet("dev", flag.ExitOnError)
		port := devFlags.String("port", "8080", "Port to listen on")
		configPath := devFlags.String("config", "", "Path to configuration file")
		rootDir := devFlags.String("root", ".", "Root directory for file discovery")

		// Parse the dev subcommand flags
		devFlags.Parse(os.Args[2:])

		// Resolve root directory to absolute path
		absRootDir, err := filepath.Abs(*rootDir)
		if err != nil {
			log.Errorf("Failed to resolve root directory: %v", err)
			os.Exit(1)
		}

		StartDevServer(*port, log, *configPath, absRootDir)
		return
	}

	// Check if we should start LSP server
	if len(os.Args) > 1 && os.Args[1] == "lsp" {
		// Parse flags for LSP server
		lspFlags := flag.NewFlagSet("lsp", flag.ExitOnError)
		port := lspFlags.String("port", "8081", "Port to listen on")
		logLevel := lspFlags.String("log-level", "info", "Log level (trace, debug, info, warn, error)")

		// Parse the lsp subcommand flags
		lspFlags.Parse(os.Args[2:])

		// Set log level
		switch *logLevel {
		case "trace":
			log.SetLevel(logrus.TraceLevel)
		case "debug":
			log.SetLevel(logrus.DebugLevel)
		case "info":
			log.SetLevel(logrus.InfoLevel)
		case "warn":
			log.SetLevel(logrus.WarnLevel)
		case "error":
			log.SetLevel(logrus.ErrorLevel)
		default:
			log.SetLevel(logrus.InfoLevel)
		}

		lsp.StartLSPServer(*port, log)
		return
	}

	args := compiler.ParseArgs(log)

	p := compiler.New(args, log)

	if args.FilePath == "" {
		log.Error(fmt.Errorf("no input file path provided"))
		os.Exit(1)
	}

	// Set log level based on args.LogLevel
	switch args.LogLevel {
	case "trace":
		log.SetLevel(logrus.TraceLevel)
	case "debug":
		log.SetLevel(logrus.DebugLevel)
	case "info":
		log.SetLevel(logrus.InfoLevel)
	case "warn":
		log.SetLevel(logrus.WarnLevel)
	case "error":
		log.SetLevel(logrus.ErrorLevel)
	default:
		log.SetLevel(logrus.InfoLevel)
	}

	log.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: false,
		DisableQuote:     true,
	})

	if args.Watch {
		if err := p.WatchFile(); err != nil {
			log.Error(err)
			os.Exit(1)
		}
	} else {
		code, err := p.CompileFile()
		if err != nil {
			log.Error(err)
			os.Exit(1)
		}

		outputPath := args.OutputPath
		if outputPath == "" {
			var err error
			outputPath, err = compiler.CreateTempOutputFile(*code)
			if err != nil {
				log.Error(err)
				os.Exit(1)
			}
		}

		if args.Command == "run" {
			if err := compiler.RunGoProgram(outputPath); err != nil {
				log.Error(err)
				os.Exit(1)
			}
		}
	}
}
