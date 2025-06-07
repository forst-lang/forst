package main

import (
	"fmt"
	"forst/internal/generators"
	"forst/internal/lexer"
	"forst/internal/parser"
	transformer_go "forst/internal/transformer/go"
	"forst/internal/typechecker"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
)

// Program represents the Forst compiler and its arguments.
type Program struct {
	Args   ProgramArgs
	logger *logrus.Logger
}

func NewProgram(args ProgramArgs) *Program {
	return &Program{
		Args:   args,
		logger: createLogger(),
	}
}

func (p *Program) readSourceFile() ([]byte, error) {
	source, err := os.ReadFile(p.Args.filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %v", err)
	}
	return source, nil
}

func getMemStats() runtime.MemStats {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	return mem
}

func runGoProgram(outputPath string) error {
	cmd := exec.Command("go", "run", outputPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (p *Program) reportPhase(phase string) {
	if p.Args.reportPhases {
		p.logger.Info(phase)
	}
}

// Creates a temporary directory and file for the output
func createTempOutputFile(code string) (string, error) {
	tempDir, err := os.MkdirTemp("", "forst-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %v", err)
	}
	outputPath := fmt.Sprintf("%s/main.go", tempDir)
	if err := os.WriteFile(outputPath, []byte(code), 0644); err != nil {
		return "", fmt.Errorf("failed to write temp file: %v", err)
	}
	return outputPath, nil
}

func (p *Program) compileFile() (*string, error) {
	source, err := p.readSourceFile()
	if err != nil {
		return nil, err
	}

	p.reportPhase("Performing lexical analysis...")
	memBefore := getMemStats()

	// Lexical Analysis
	l := lexer.NewLexer(source, p.Args.filePath)
	tokens := l.Lex()

	memAfter := getMemStats()
	p.logMemUsage("lexical analysis", memBefore, memAfter)

	p.debugPrintTokens(tokens)

	p.reportPhase("Performing syntax analysis...")
	memBefore = getMemStats()

	// Parsing
	psr := parser.NewParser(tokens, p.Args.filePath, p.logger)
	forstNodes, err := psr.ParseFile()
	if err != nil {
		return nil, err
	}

	memAfter = getMemStats()
	p.logMemUsage("syntax analysis", memBefore, memAfter)

	p.debugPrintForstAST(forstNodes)

	p.reportPhase("Performing semantic analysis...")
	memBefore = getMemStats()

	// Semantic Analysis
	checker := typechecker.New()

	// Collect, infer and check type
	if err := checker.CheckTypes(forstNodes); err != nil {
		p.logger.Error("Encountered error checking types: ", err)
		checker.DebugPrintCurrentScope()
		return nil, err
	}

	memAfter = getMemStats()
	p.logMemUsage("semantic analysis", memBefore, memAfter)

	p.debugPrintTypeInfo(checker)

	p.reportPhase("Performing code generation...")
	memBefore = getMemStats()

	// Transform to Go AST with type information
	transformer := transformer_go.New(checker)
	goAST, err := transformer.TransformForstFileToGo(forstNodes)
	if err != nil {
		return nil, err
	}

	p.debugPrintGoAST(goAST)
	// Generate Go code
	goCode, err := generators.GenerateGoCode(goAST)
	if err != nil {
		return nil, err
	}

	memAfter = getMemStats()
	p.logMemUsage("code generation", memBefore, memAfter)

	if p.Args.outputPath != "" {
		if err := os.WriteFile(p.Args.outputPath, []byte(goCode), 0644); err != nil {
			return nil, fmt.Errorf("error writing output file: %v", err)
		}
	} else if p.Args.trace {
		p.logger.Info("Generated Go code:")
		fmt.Println(goCode)
	}

	return &goCode, nil
}

func (p *Program) watchFile() error {
	// Create a new watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("error creating watcher: %v", err)
	}
	defer func() {
		if err := watcher.Close(); err != nil {
			p.logger.Errorf("Error closing watcher: %v", err)
		}
	}()

	// Start watching the file
	if err := watcher.Add(p.Args.filePath); err != nil {
		return fmt.Errorf("error watching file: %v", err)
	}

	p.logger.Infof("Watching %s for changes...", p.Args.filePath)

	// Initial compilation
	if code, err := p.compileFile(); err != nil {
		p.logger.Error(err)
		p.logger.Warn("Not running program because of errors during compilation")
	} else {
		outputPath := p.Args.outputPath
		if outputPath == "" {
			var err error
			outputPath, err = createTempOutputFile(*code)
			if err != nil {
				p.logger.Error(err)
				os.Exit(1)
			}
		}

		// Run the compiled program
		if err := runGoProgram(outputPath); err != nil {
			p.logger.Error(err)
		}
	}

	// Create a debounce timer
	var debounceTimer *time.Timer

	// Watch for changes
	for {
		select {
		case event := <-watcher.Events:
			if event.Op&fsnotify.Write == fsnotify.Write {
				// Cancel any existing timer
				if debounceTimer != nil {
					debounceTimer.Stop()
				}

				// Create a new timer
				debounceTimer = time.AfterFunc(100*time.Millisecond, func() {
					p.logger.Info("File changed, recompiling...")
					if code, err := p.compileFile(); err != nil {
						p.logger.Error(err)
						p.logger.Warn("Not running program because of errors during compilation")
					} else {
						outputPath := p.Args.outputPath
						if outputPath == "" {
							var err error
							outputPath, err = createTempOutputFile(*code)
							if err != nil {
								p.logger.Error(err)
								os.Exit(1)
							}
						}
						// Run the compiled program
						if err := runGoProgram(outputPath); err != nil {
							p.logger.Error(err)
						}
					}
				})
			}
		case err := <-watcher.Errors:
			p.logger.Error("Error watching file:", err)
		}
	}
}

func createLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "15:04:05.000",
	})
	return logger
}
