package main

import (
	"os"
	"testing"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		want     ProgramArgs
		wantHelp bool
	}{
		{
			name: "run command with file",
			args: []string{"forst", "run", "test.ft"},
			want: ProgramArgs{
				command:  "run",
				filePath: "test.ft",
			},
			wantHelp: false,
		},
		{
			name: "build command with file",
			args: []string{"forst", "build", "test.ft"},
			want: ProgramArgs{
				command:  "build",
				filePath: "test.ft",
			},
			wantHelp: false,
		},
		{
			name: "run with debug flag",
			args: []string{"forst", "run", "-debug", "test.ft"},
			want: ProgramArgs{
				command:  "run",
				filePath: "test.ft",
				debug:    true,
			},
			wantHelp: false,
		},
		{
			name: "run with watch and output",
			args: []string{"forst", "run", "-watch", "-o", "output.go", "test.ft"},
			want: ProgramArgs{
				command:    "run",
				filePath:   "test.ft",
				watch:      true,
				outputPath: "output.go",
			},
			wantHelp: false,
		},
		{
			name:     "help flag",
			args:     []string{"forst", "--help"},
			want:     ProgramArgs{},
			wantHelp: true,
		},
		{
			name:     "invalid command",
			args:     []string{"forst", "invalid", "test.ft"},
			want:     ProgramArgs{},
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

			got := ParseArgs()

			// Compare fields
			if got.command != tt.want.command {
				t.Errorf("ParseArgs().command = %v, want %v", got.command, tt.want.command)
			}
			if got.filePath != tt.want.filePath {
				t.Errorf("ParseArgs().filePath = %v, want %v", got.filePath, tt.want.filePath)
			}
			if got.debug != tt.want.debug {
				t.Errorf("ParseArgs().debug = %v, want %v", got.debug, tt.want.debug)
			}
			if got.watch != tt.want.watch {
				t.Errorf("ParseArgs().watch = %v, want %v", got.watch, tt.want.watch)
			}
			if got.outputPath != tt.want.outputPath {
				t.Errorf("ParseArgs().outputPath = %v, want %v", got.outputPath, tt.want.outputPath)
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
			got := ParseArgs()

			if got != (ProgramArgs{}) {
				t.Errorf("ParseArgs() = %v, want empty ProgramArgs for invalid args", got)
			}
		})
	}
}
