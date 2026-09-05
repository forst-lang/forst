package testutil

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// FixtureMeta describes one testdata fixture directory (meta.json).
type FixtureMeta struct {
	ID     string `json:"id"`
	Expect string `json:"expect"`
	Title  string `json:"title"`
	Code   string `json:"code"`
	Matrix bool   `json:"matrix"`
}

// Fixture is a loaded meta.json + input.ft pair.
type Fixture struct {
	Dir  string
	Meta FixtureMeta
	Src  string
}

// LoadFixtureDir loads meta.json + input.ft from a fixture directory.
func LoadFixtureDir(dir string) (Fixture, error) {
	metaBytes, err := os.ReadFile(filepath.Join(dir, "meta.json"))
	if err != nil {
		return Fixture{}, err
	}
	var m FixtureMeta
	if err := json.Unmarshal(metaBytes, &m); err != nil {
		return Fixture{}, fmt.Errorf("%s/meta.json: %w", dir, err)
	}
	src, err := os.ReadFile(filepath.Join(dir, "input.ft"))
	if err != nil {
		return Fixture{}, err
	}
	if m.ID == "" {
		m.ID = filepath.Base(dir)
	}
	return Fixture{Dir: dir, Meta: m, Src: string(src)}, nil
}

// ListFixtureDirs walks root for directories containing meta.json + input.ft.
func ListFixtureDirs(root string) ([]Fixture, error) {
	var out []Fixture
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || info.Name() != "meta.json" {
			return nil
		}
		fx, err := LoadFixtureDir(filepath.Dir(path))
		if err != nil {
			return err
		}
		out = append(out, fx)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Meta.ID < out[j].Meta.ID })
	return out, nil
}

// TestdataDir returns <caller-package>/testdata/<rel...>.
// Pass rel segments such as "refinements" or "refinements", "clobber".
func TestdataDir(tb testing.TB, rel ...string) string {
	tb.Helper()
	_, file, _, ok := runtime.Caller(1)
	if !ok {
		tb.Fatal("runtime.Caller failed")
	}
	parts := append([]string{filepath.Dir(file), "testdata"}, rel...)
	return filepath.Join(parts...)
}

// SanitizeTestName maps a fixture id to a valid subtest name.
func SanitizeTestName(id string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-', r == '/':
			return r
		default:
			return '_'
		}
	}, id)
}
