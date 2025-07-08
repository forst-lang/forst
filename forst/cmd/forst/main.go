// Package main is the main package for the forst compiler.
package main

import (
	"flag"
	"fmt"
	"forst/cmd/forst/compiler"
	"os"

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
		port := flag.String("port", "8080", "Port to listen on")
		flag.Parse()

		StartDevServer(*port, log)
		return
	}

	args := compiler.ParseArgs(log)

	p := compiler.New(args, log)

	if args.FilePath == "" {
		log.Error(fmt.Errorf("no input file path provided"))
		os.Exit(1)
	}

	if args.Trace {
		log.SetLevel(logrus.TraceLevel)
	} else if args.Debug {
		log.SetLevel(logrus.DebugLevel)
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
