// Package main is the main package for the forst compiler.
package main

import (
	"fmt"
	"forst/cmd/forst/compiler"
	"forst/internal/logger"
	"os"

	logrus "github.com/sirupsen/logrus"
)

func main() {
	log := logger.New()

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
