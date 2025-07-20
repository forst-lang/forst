// Package main is the main package for the forst compiler.
package main

import (
	"encoding/json"
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
	// Version is the current version of Forst
	Version = "dev"
	// Commit is the git commit hash
	Commit = "unknown"
	// Date is the build date
	Date = "unknown"
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

		// Set version information in LSP package
		lsp.Version = Version
		lsp.Commit = Commit
		lsp.Date = Date

		lsp.StartLSPServer(*port, log)
		return
	}

	// Check if we should dump debug info
	if len(os.Args) > 1 && os.Args[1] == "dump" {
		// Parse flags for dump command
		dumpFlags := flag.NewFlagSet("dump", flag.ExitOnError)
		filePath := dumpFlags.String("file", "", "Path to Forst file to dump")
		compression := dumpFlags.Bool("compression", true, "Enable compression for debug output")
		format := dumpFlags.String("format", "json", "Output format (json, pretty)")
		phase := dumpFlags.String("phase", "all", "Specific phase to dump (lexer, parser, typechecker, transformer, all)")

		// Parse the dump subcommand flags
		dumpFlags.Parse(os.Args[2:])

		if *filePath == "" {
			log.Error("dump command requires --file flag")
			os.Exit(1)
		}

		// Set version information in LSP package
		lsp.Version = Version
		lsp.Commit = Commit
		lsp.Date = Date

		handleDumpCommand(*filePath, *compression, *format, *phase, log)
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

// handleDumpCommand dumps debug information for a Forst file using LSP functionality
func handleDumpCommand(filePath string, compression bool, format string, phase string, log *logrus.Logger) {
	// Create a temporary LSP server instance for dumping
	server := lsp.NewLSPServer(":0", log) // Port 0 means we won't actually listen

	// Read the file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		log.Errorf("Failed to read file %s: %v", filePath, err)
		os.Exit(1)
	}

	// Convert file path to URI format
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		log.Errorf("Failed to get absolute path: %v", err)
		os.Exit(1)
	}
	uri := "file://" + absPath

	// Create a mock LSP request to simulate debugInfo call
	request := lsp.LSPRequest{
		JSONRPC: "2.0",
		ID:      "dump",
		Method:  "textDocument/debugInfo",
		Params:  nil, // Will be set below
	}

	// Create params with compression setting
	params := map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri": uri,
		},
		"compression": compression,
	}

	// Marshal params to JSON
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		log.Errorf("Failed to marshal params: %v", err)
		os.Exit(1)
	}
	request.Params = paramsJSON

	// Call the debugInfo handler directly
	response := server.HandleDebugInfoDirect(request, string(content))

	// Format the output
	var output []byte
	if format == "pretty" {
		output, err = json.MarshalIndent(response.Result, "", "  ")
	} else {
		output, err = json.Marshal(response.Result)
	}

	if err != nil {
		log.Errorf("Failed to marshal output: %v", err)
		os.Exit(1)
	}

	// Print the output
	fmt.Println(string(output))
}
