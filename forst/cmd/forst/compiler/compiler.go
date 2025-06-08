package compiler

import (
	"fmt"
	"forst/internal/generators"
	"forst/internal/lexer"
	"forst/internal/logger"
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

// Compiler represents the Forst compiler and its arguments.
type Compiler struct {
	Args Args
	log  *logrus.Logger
}

func New(args Args, log *logrus.Logger) *Compiler {
	if log == nil {
		log = logger.New()
	}

	return &Compiler{
		Args: args,
		log:  log,
	}
}

func (c *Compiler) readSourceFile() ([]byte, error) {
	source, err := os.ReadFile(c.Args.FilePath)
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

func RunGoProgram(outputPath string) error {
	cmd := exec.Command("go", "run", outputPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (c *Compiler) reportPhase(phase string) {
	if c.Args.ReportPhases {
		c.log.Info(phase)
	}
}

// CreateTempOutputFile creates a temporary directory and file for the output
func CreateTempOutputFile(code string) (string, error) {
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

// CompileFile compiles a Forst file and returns the Go code
func (c *Compiler) CompileFile() (*string, error) {
	source, err := c.readSourceFile()
	if err != nil {
		return nil, err
	}

	c.reportPhase("Performing lexical analysis...")
	memBefore := getMemStats()

	// Lexical Analysis
	l := lexer.New(source, c.Args.FilePath, c.log)
	tokens := l.Lex()

	memAfter := getMemStats()
	c.LogMemUsage("lexical analysis", memBefore, memAfter)

	c.DebugPrintTokens(tokens)

	c.reportPhase("Performing syntax analysis...")
	memBefore = getMemStats()

	// Parsing
	psr := parser.New(tokens, c.Args.FilePath, c.log)
	forstNodes, err := psr.ParseFile()
	if err != nil {
		return nil, err
	}

	memAfter = getMemStats()
	c.LogMemUsage("syntax analysis", memBefore, memAfter)

	c.DebugPrintForstAST(forstNodes)

	c.reportPhase("Performing semantic analysis...")
	memBefore = getMemStats()

	// Semantic Analysis
	checker := typechecker.New(c.log)

	// Collect, infer and check type
	if err := checker.CheckTypes(forstNodes); err != nil {
		c.log.Error("Encountered error checking types: ", err)
		checker.DebugPrintCurrentScope()
		return nil, err
	}

	memAfter = getMemStats()
	c.LogMemUsage("semantic analysis", memBefore, memAfter)

	c.debugPrintTypeInfo(checker)

	c.reportPhase("Performing code generation...")
	memBefore = getMemStats()

	// Transform to Go AST with type information
	transformer := transformer_go.New(checker, c.log)
	goAST, err := transformer.TransformForstFileToGo(forstNodes)
	if err != nil {
		return nil, err
	}

	c.debugPrintGoAST(goAST)
	// Generate Go code
	goCode, err := generators.GenerateGoCode(goAST)
	if err != nil {
		return nil, err
	}

	memAfter = getMemStats()
	c.LogMemUsage("code generation", memBefore, memAfter)

	if c.Args.OutputPath != "" {
		if err := os.WriteFile(c.Args.OutputPath, []byte(goCode), 0644); err != nil {
			return nil, fmt.Errorf("error writing output file: %v", err)
		}
	} else if c.Args.Trace {
		c.log.Info("Generated Go code:")
		fmt.Println(goCode)
	}

	return &goCode, nil
}

// WatchFile watches the Forst file for changes and recompiles it
func (c *Compiler) WatchFile() error {
	// Create a new watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("error creating watcher: %v", err)
	}
	defer func() {
		if err := watcher.Close(); err != nil {
			c.log.Errorf("Error closing watcher: %v", err)
		}
	}()

	// Start watching the file
	if err := watcher.Add(c.Args.FilePath); err != nil {
		return fmt.Errorf("error watching file: %v", err)
	}

	c.log.Infof("Watching %s for changes...", c.Args.FilePath)

	// Initial compilation
	if code, err := c.CompileFile(); err != nil {
		c.log.Error(err)
		c.log.Warn("Not running program because of errors during compilation")
	} else {
		outputPath := c.Args.OutputPath
		if outputPath == "" {
			var err error
			outputPath, err = CreateTempOutputFile(*code)
			if err != nil {
				c.log.Error(err)
				os.Exit(1)
			}
		}

		// Run the compiled program
		if err := RunGoProgram(outputPath); err != nil {
			c.log.Error(err)
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
					c.log.Info("File changed, recompiling...")
					if code, err := c.CompileFile(); err != nil {
						c.log.Error(err)
						c.log.Warn("Not running program because of errors during compilation")
					} else {
						outputPath := c.Args.OutputPath
						if outputPath == "" {
							var err error
							outputPath, err = CreateTempOutputFile(*code)
							if err != nil {
								c.log.Error(err)
								os.Exit(1)
							}
						}
						// Run the compiled program
						if err := RunGoProgram(outputPath); err != nil {
							c.log.Error(err)
						}
					}
				})
			}
		case err := <-watcher.Errors:
			c.log.Error("Error watching file:", err)
		}
	}
}
