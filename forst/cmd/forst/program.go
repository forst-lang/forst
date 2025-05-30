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
	"path/filepath"
	"runtime"
	"time"

	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
)

type Program struct {
	Args ProgramArgs
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
		log.Info(phase)
	}
}

func (p *Program) compileFile() (*string, error) {
	source, err := p.readSourceFile()
	if err != nil {
		return nil, err
	}

	p.reportPhase("Performing lexical analysis...")
	memBefore := getMemStats()

	// Lexical Analysis
	tokens := lexer.Lexer(source, lexer.Context{FilePath: p.Args.filePath})

	memAfter := getMemStats()
	p.logMemUsage("lexical analysis", memBefore, memAfter)

	p.debugPrintTokens(tokens)

	p.reportPhase("Performing syntax analysis...")
	memBefore = getMemStats()

	// Parsing
	forstNodes, err := parser.NewParser(tokens, p.Args.filePath).ParseFile()
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
		fmt.Println(goCode)
	}

	return &goCode, nil
}

func (p *Program) watchFile() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer func() {
		if err := watcher.Close(); err != nil {
			log.Printf("Error closing watcher: %v", err)
		}
	}()

	// Watch the directory containing the file to catch renames
	dir := filepath.Dir(p.Args.filePath)
	if err := watcher.Add(dir); err != nil {
		return err
	}

	log.Infof("Watching %s for changes...", p.Args.filePath)

	// Initial compilation
	if code, err := p.compileFile(); err != nil {
		log.Error(err)
		log.Warn("Not running program because of errors during compilation")
	} else {
		outputPath := p.Args.outputPath
		if outputPath == "" {
			// Create temp directory if needed
			tempDir, err := os.MkdirTemp("", "forst-*")
			if err != nil {
				log.Error(err)
				os.Exit(1)
			}
			outputPath = fmt.Sprintf("%s/main.go", tempDir)
			if err := os.WriteFile(outputPath, []byte(*code), 0644); err != nil {
				log.Error(err)
				os.Exit(1)
			}
		}

		// Run the compiled program
		if err := runGoProgram(outputPath); err != nil {
			log.Error(err)
		}
	}

	// Debounce timer to avoid multiple compilations for rapid changes
	var debounceTimer *time.Timer
	for {
		select {
		case event := <-watcher.Events:
			if event.Name == p.Args.filePath {
				if debounceTimer != nil {
					debounceTimer.Stop()
				}
				debounceTimer = time.AfterFunc(100*time.Millisecond, func() {
					log.Info("File changed, recompiling...")
					code, err := p.compileFile()
					if err != nil {
						log.Error(err)
						log.Warn("Not running program because of errors during compilation")
					} else {
						outputPath := p.Args.outputPath
						if outputPath == "" {
							// Create temp directory if needed
							tempDir, err := os.MkdirTemp("", "forst-*")
							if err != nil {
								log.Error(err)
								os.Exit(1)
							}
							outputPath = fmt.Sprintf("%s/main.go", tempDir)
							if err := os.WriteFile(outputPath, []byte(*code), 0644); err != nil {
								log.Error(err)
								os.Exit(1)
							}
						}
						// Run the compiled program
						if err := runGoProgram(p.Args.outputPath); err != nil {
							log.Error(err)
						}
					}
				})
			}
		case err := <-watcher.Errors:
			return err
		}
	}
}
