package main

import (
	"flag"
	"fmt"
	"io"
	iofs "io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"forst/internal/printer"

	"github.com/sirupsen/logrus"
)

// runFmtCommand implements `forst fmt`, modeled on `go fmt` / gofmt: format .ft files in place by
// default, with -l to list files that differ (without writing) and -n for a dry run (no writes).
func runFmtCommand(argv []string, log *logrus.Logger, out io.Writer) error {
	flags := flag.NewFlagSet("fmt", flag.ExitOnError)
	flags.SetOutput(io.Discard)
	list := flags.Bool("l", false, "list files whose formatting differs (do not write)")
	dry := flags.Bool("n", false, "dry run: do not write; print write <path> for each file that would change")
	tabWidth := flags.Int("tabwidth", 8, "tab width when using -spaces to expand leading tabs")
	useSpaces := flags.Bool("spaces", false, "expand leading tabs to spaces (-tabwidth per tab)")
	if err := flags.Parse(argv); err != nil {
		return err
	}
	paths := flags.Args()
	if len(paths) == 0 {
		paths = []string{"."}
	}
	files, err := collectFtPaths(paths)
	if err != nil {
		return err
	}
	sort.Strings(files)

	for _, path := range files {
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		src := string(b)
		formatted := printer.FormatDocument(src, path, *tabWidth, *useSpaces, log)
		if formatted == src {
			continue
		}
		switch {
		case *list:
			_, _ = fmt.Fprintln(out, path)
		case *dry:
			_, _ = fmt.Fprintf(out, "write %s\n", path)
		default:
			info, statErr := os.Stat(path)
			perm := iofs.FileMode(0o644)
			if statErr == nil {
				perm = info.Mode() & iofs.ModePerm
			}
			if err := os.WriteFile(path, []byte(formatted), perm); err != nil {
				return err
			}
		}
	}
	return nil
}

func collectFtPaths(paths []string) ([]string, error) {
	var out []string
	for _, p := range paths {
		p = filepath.Clean(p)
		fi, err := os.Stat(p)
		if err != nil {
			return nil, err
		}
		if !fi.IsDir() {
			if strings.HasSuffix(p, ".ft") {
				out = append(out, p)
			}
			continue
		}
		err = filepath.WalkDir(p, func(path string, d iofs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				return nil
			}
			if strings.HasSuffix(path, ".ft") {
				out = append(out, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}
