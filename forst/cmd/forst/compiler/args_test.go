package compiler

import (
	"forst/internal/logger"
	"os"
	"testing"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		want     Args
		wantHelp bool
	}{
		{
			name: "run command with file",
			args: []string{"forst", "run", "test.ft"},
			want: Args{
				Command:  "run",
				FilePath: "test.ft",
			},
			wantHelp: false,
		},
		{
			name: "build command with file",
			args: []string{"forst", "build", "test.ft"},
			want: Args{
				Command:  "build",
				FilePath: "test.ft",
			},
			wantHelp: false,
		},
		{
			name: "run with debug flag",
			args: []string{"forst", "run", "-debug", "test.ft"},
			want: Args{
				Command:  "run",
				FilePath: "test.ft",
				Debug:    true,
			},
			wantHelp: false,
		},
		{
			name: "run with watch and output",
			args: []string{"forst", "run", "-watch", "-o", "output.go", "test.ft"},
			want: Args{
				Command:    "run",
				FilePath:   "test.ft",
				Watch:      true,
				OutputPath: "output.go",
			},
			wantHelp: false,
		},
		{
			name:     "help flag",
			args:     []string{"forst", "--help"},
			want:     Args{},
			wantHelp: true,
		},
		{
			name:     "invalid command",
			args:     []string{"forst", "invalid", "test.ft"},
			want:     Args{},
			wantHelp: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original args and restore after test
			origArgs := os.Args
			defer func() { os.Args = origArgs }()

			os.Args = tt.args

			if tt.wantHelp {
				// Help flag should cause os.Exit(0)
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("Expected ParseArgs() to exit with status 0 for help flag")
					}
				}()
			}

			log := logger.New()
			got := ParseArgs(log)

			// Compare fields
			if got.Command != tt.want.Command {
				t.Errorf("ParseArgs().command = %v, want %v", got.Command, tt.want.Command)
			}
			if got.FilePath != tt.want.FilePath {
				t.Errorf("ParseArgs().filePath = %v, want %v", got.FilePath, tt.want.FilePath)
			}
			if got.Debug != tt.want.Debug {
				t.Errorf("ParseArgs().debug = %v, want %v", got.Debug, tt.want.Debug)
			}
			if got.Watch != tt.want.Watch {
				t.Errorf("ParseArgs().watch = %v, want %v", got.Watch, tt.want.Watch)
			}
			if got.OutputPath != tt.want.OutputPath {
				t.Errorf("ParseArgs().outputPath = %v, want %v", got.OutputPath, tt.want.OutputPath)
			}
		})
	}
}

func TestInvalidArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "no arguments",
			args: []string{"forst"},
		},
		{
			name: "no file specified",
			args: []string{"forst", "run"},
		},
		{
			name: "build with watch",
			args: []string{"forst", "build", "-watch", "test.ft"},
		},
		{
			name: "watch without output",
			args: []string{"forst", "run", "-watch", "test.ft"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original args and restore after test
			origArgs := os.Args
			defer func() { os.Args = origArgs }()

			os.Args = tt.args
			log := logger.New()
			got := ParseArgs(log)

			if got != (Args{}) {
				t.Errorf("ParseArgs() = %v, want empty ProgramArgs for invalid args", got)
			}
		})
	}
}
