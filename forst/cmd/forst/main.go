// Package main is the main package for the forst compiler.
package main

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
)

func main() {
	args := ParseArgs()
	p := New(args)

	if args.filePath == "" {
		p.log.Error(fmt.Errorf("no input file path provided"))
		os.Exit(1)
	}

	if args.trace {
		p.log.SetLevel(log.TraceLevel)
	} else if args.debug {
		p.log.SetLevel(log.DebugLevel)
	}

	p.log.SetFormatter(&log.TextFormatter{
		DisableTimestamp: false,
		DisableQuote:     true,
	})

	if args.watch {
		if err := p.watchFile(); err != nil {
			p.log.Error(err)
			os.Exit(1)
		}
	} else {
		code, err := p.compileFile()
		if err != nil {
			p.log.Error(err)
			os.Exit(1)
		}

		outputPath := args.outputPath
		if outputPath == "" {
			var err error
			outputPath, err = createTempOutputFile(*code)
			if err != nil {
				p.log.Error(err)
				os.Exit(1)
			}
		}

		if args.command == "run" {
			if err := runGoProgram(outputPath); err != nil {
				p.log.Error(err)
				os.Exit(1)
			}
		}
	}
}
