package compiler

import (
	"fmt"
	"forst/internal/logger"
	"os"
	"os/exec"
	"runtime"

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
