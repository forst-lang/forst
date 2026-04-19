// Package main is the main package for the forst compiler.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"forst/cmd/forst/lsp"
	"forst/internal/compiler"
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
	os.Exit(runMain(os.Args))
}

// runMain contains the forst CLI entry logic. It returns a process exit code (0 = success).
func runMain(argv []string) int {
	if len(argv) > 1 && (argv[1] == "version" || argv[1] == "--version" || argv[1] == "-v") {
		printVersionInfo()
		return 0
	}

	log := newLogger()

	// Check if we should start dev server
	if len(argv) > 1 && argv[1] == "dev" {
		// Parse flags for dev server
		devFlags := flag.NewFlagSet("dev", flag.ExitOnError)
		port := devFlags.String("port", "8080", "Port to listen on")
		configPath := devFlags.String("config", "", "Path to configuration file")
		rootDir := devFlags.String("root", ".", "Root directory for file discovery")
		logLevel := devFlags.String("log-level", "info", "Log level (trace, debug, info, warn, error)")

		// Parse the dev subcommand flags
		if err := devFlags.Parse(argv[2:]); err != nil {
			log.Errorf("dev flags: %v", err)
			return 1
		}

		// Resolve root directory to absolute path
		absRootDir, err := filepath.Abs(*rootDir)
		if err != nil {
			log.Errorf("Failed to resolve root directory: %v", err)
			return 1
		}

		if err := StartDevServer(*port, log, *configPath, absRootDir, logLevel); err != nil {
			return 1
		}
		return 0
	}

	if len(argv) > 1 && argv[1] == "fmt" {
		if err := runFmtCommand(argv[2:], log, os.Stdout); err != nil {
			log.Error(err)
			return 1
		}
		return 0
	}

	// Check if we should generate TypeScript client
	if len(argv) > 1 && argv[1] == "generate" {
		if err := generateCommand(argv[2:]); err != nil {
			log.Error(err)
			return 1
		}
		return 0
	}

	// Check if we should start LSP server
	if len(argv) > 1 && argv[1] == "lsp" {
		// Parse flags for LSP server
		lspFlags := flag.NewFlagSet("lsp", flag.ExitOnError)
		port := lspFlags.String("port", "8081", "Port to listen on")
		logLevel := lspFlags.String("log-level", "info", "Log level (trace, debug, info, warn, error)")

		// Parse the lsp subcommand flags
		if err := lspFlags.Parse(argv[2:]); err != nil {
			log.Errorf("lsp flags: %v", err)
			return 1
		}

		// Set log level
		setLogLevel(log, *logLevel)

		// Set version information in LSP package
		lsp.Version = Version
		lsp.Commit = Commit
		lsp.Date = Date

		if err := lsp.StartLSPServer(*port, log); err != nil {
			return 1
		}
		return 0
	}

	// Check if we should dump debug info
	if len(argv) > 1 && argv[1] == "dump" {
		// Parse flags for dump command
		dumpFlags := flag.NewFlagSet("dump", flag.ExitOnError)
		filePath := dumpFlags.String("file", "", "Path to Forst file to dump")
		compression := dumpFlags.Bool("compression", false, "Enable compression for debug output")
		format := dumpFlags.String("format", "json", "Output format (json, pretty)")
		phase := dumpFlags.String("phase", "all", "Specific phase to dump (lexer, parser, typechecker, transformer, all)")
		summary := dumpFlags.Bool("summary", false, "Show only phase summaries")

		// Parse the dump subcommand flags
		if err := dumpFlags.Parse(argv[2:]); err != nil {
			log.Errorf("dump flags: %v", err)
			return 1
		}

		if *filePath == "" {
			log.Error("dump command requires --file flag")
			return 1
		}

		// Set version information in LSP package
		lsp.Version = Version
		lsp.Commit = Commit
		lsp.Date = Date

		handleDumpCommand(*filePath, *compression, *format, *phase, *summary, log)
		return 0
	}

	saved := os.Args
	os.Args = argv
	defer func() { os.Args = saved }()

	args := compiler.ParseArgs(log)

	p := compiler.New(args, log)

	if args.FilePath == "" {
		log.Error(fmt.Errorf("no input file path provided"))
		return 1
	}

	// Set log level based on args.LogLevel
	setLogLevel(log, args.LogLevel)

	log.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: false,
		DisableQuote:     true,
	})

	if args.Watch {
		if err := p.WatchFile(); err != nil {
			log.Error(err)
			return 1
		}
	} else {
		code, err := p.CompileFile()
		if err != nil {
			log.Error(err)
			return 1
		}

		outputPath := args.OutputPath
		if outputPath == "" {
			var err error
			outputPath, err = compiler.CreateTempOutputFile(*code)
			if err != nil {
				log.Error(err)
				return 1
			}
		}

		if args.Command == "run" {
			if err := compiler.RunGoProgram(outputPath); err != nil {
				log.Error(err)
				return 1
			}
		}
	}
	return 0
}

func newLogger() *logrus.Logger {
	logger := logrus.New()
	if Version == "dev" {
		logger.SetLevel(logrus.DebugLevel)
		return logger
	}
	logger.SetLevel(logrus.InfoLevel)
	return logger
}

func setLogLevel(log *logrus.Logger, level string) {
	switch level {
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
}

// handleDumpCommand dumps debug information for a Forst file using LSP functionality
func handleDumpCommand(filePath string, compression bool, format string, _ string, summary bool, log *logrus.Logger) {
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

	// Create params with all settings
	params := map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri": uri,
		},
		"compression": compression,
		"summary":     summary,
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
