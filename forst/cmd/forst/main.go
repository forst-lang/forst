package main

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
)

func main() {
	args := ParseArgs()
	program := Program{
		Args: args,
	}

	if args.filePath == "" {
		log.Error(fmt.Errorf("no input file path provided"))
		os.Exit(1)
	}

	if args.trace {
		log.SetLevel(log.TraceLevel)
	} else if args.debug {
		log.SetLevel(log.DebugLevel)
	}

	log.SetFormatter(&log.TextFormatter{
		DisableTimestamp: false,
		DisableQuote:     true,
	})

	if args.watch {
		if err := program.watchFile(); err != nil {
			log.Error(err)
			os.Exit(1)
		}
	} else {
		if err := program.compileFile(); err != nil {
			log.Error(err)
			os.Exit(1)
		}

		if args.command == "run" {
			if err := runGoProgram(args.outputPath); err != nil {
				log.Error(err)
				os.Exit(1)
			}
		}
	}
}
