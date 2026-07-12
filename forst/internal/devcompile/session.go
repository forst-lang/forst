package devcompile

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"forst/internal/ast"
	"forst/internal/forstpkg"
	"forst/internal/modulecheck"

	"github.com/sirupsen/logrus"
)

// FileFingerprint identifies a source file by path and content metadata.
type FileFingerprint struct {
	Path  string
	Mtime int64
	Size  int64
}

// Session is a process-lifetime dev reload cache keyed by file fingerprints.
type Session struct {
	boundaryRoot string
	mu           sync.Mutex
	fileCache    map[string]fileEntry
	modResult    *modulecheck.ModuleResult
	modRoot      string
	modPrint     string
	dirty        bool
}

type fileEntry struct {
	fp    FileFingerprint
	nodes []ast.Node
}

// NewSession creates a dev compile cache for boundaryRoot.
func NewSession(boundaryRoot string) *Session {
	return &Session{
		boundaryRoot: filepath.Clean(boundaryRoot),
		fileCache:    make(map[string]fileEntry),
	}
}

// NoteChange invalidates caches after a source file change.
func (s *Session) NoteChange(path string) {
	if s == nil || path == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	abs, err := filepath.Abs(path)
	if err != nil {
		s.dirty = true
		delete(s.fileCache, path)
		return
	}
	delete(s.fileCache, abs)
	s.dirty = true
}

// ParseFile returns cached or freshly parsed AST for path.
func (s *Session) ParseFile(log *logrus.Logger, path string) ([]ast.Node, error) {
	if s == nil {
		return forstpkg.ParseForstFile(log, path)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	fp, err := fingerprintFile(abs)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	if ent, ok := s.fileCache[abs]; ok && ent.fp == fp {
		nodes := ent.nodes
		s.mu.Unlock()
		return nodes, nil
	}
	s.mu.Unlock()

	nodes, err := forstpkg.ParseForstFile(log, abs)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.fileCache[abs] = fileEntry{fp: fp, nodes: nodes}
	s.mu.Unlock()
	return nodes, nil
}

// ParseAndMerge parses paths with per-file caching and merges in sorted order.
func (s *Session) ParseAndMerge(log *logrus.Logger, paths []string) ([]ast.Node, map[string][]ast.Node, error) {
	if s == nil {
		return forstpkg.ParseAndMergePackage(log, paths)
	}
	sorted := append([]string(nil), paths...)
	sort.Strings(sorted)
	byPath := make(map[string][]ast.Node, len(sorted))
	var lists [][]ast.Node
	for _, p := range sorted {
		nodes, err := s.ParseFile(log, p)
		if err != nil {
			return nil, nil, fmt.Errorf("parse %s: %w", p, err)
		}
		byPath[p] = nodes
		lists = append(lists, nodes)
	}
	return forstpkg.MergePackageASTs(lists), byPath, nil
}

// ParsedFilesForModule returns cached parses whose fingerprints still match disk.
func (s *Session) ParsedFilesForModule(moduleRoot string) (map[string][]ast.Node, bool) {
	if s == nil {
		return nil, false
	}
	_ = moduleRoot
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.fileCache) == 0 {
		return nil, false
	}
	out := make(map[string][]ast.Node)
	for path, ent := range s.fileCache {
		fp, err := fingerprintFile(path)
		if err != nil || fp != ent.fp {
			continue
		}
		out[path] = ent.nodes
	}
	return out, len(out) > 0
}

// CachedModuleResult returns the last module result when the module is not dirty.
func (s *Session) CachedModuleResult(moduleRoot string) (*modulecheck.ModuleResult, bool) {
	if s == nil {
		return nil, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.dirty || s.modResult == nil || s.modRoot != filepath.Clean(moduleRoot) {
		return nil, false
	}
	return s.modResult, true
}

// StoreModuleResult records a successful modulecheck and clears the dirty flag.
func (s *Session) StoreModuleResult(moduleRoot string, result *modulecheck.ModuleResult) {
	if s == nil || result == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.modRoot = filepath.Clean(moduleRoot)
	s.modResult = result
	s.modPrint = moduleFingerprint(result)
	s.dirty = false
}

func fingerprintFile(path string) (FileFingerprint, error) {
	st, err := os.Stat(path)
	if err != nil {
		return FileFingerprint{}, err
	}
	return FileFingerprint{
		Path:  path,
		Mtime: st.ModTime().UnixNano(),
		Size:  st.Size(),
	}, nil
}

func moduleFingerprint(result *modulecheck.ModuleResult) string {
	if result == nil {
		return ""
	}
	var paths []string
	for _, files := range result.ForstPkgToFiles {
		for _, p := range files {
			paths = append(paths, p)
		}
	}
	sort.Strings(paths)
	h := sha256.New()
	for _, p := range paths {
		fp, err := fingerprintFile(p)
		if err != nil {
			h.Write([]byte("err:" + p))
			continue
		}
		h.Write([]byte(fmt.Sprintf("%s:%d:%d\n", fp.Path, fp.Mtime, fp.Size)))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// CollectForstImportLocals scans nodes for Forst sibling import local names.
func CollectForstImportLocals(nodes []ast.Node) []string {
	seen := make(map[string]struct{})
	var out []string
	add := func(imp ast.ImportNode) {
		local := imp.Path
		if imp.Alias != nil {
			local = string(imp.Alias.ID)
		} else if strings.Contains(imp.Path, "/") {
			return
		}
		if local == "" {
			return
		}
		if _, ok := seen[local]; ok {
			return
		}
		seen[local] = struct{}{}
		out = append(out, local)
	}
	for _, node := range nodes {
		switch n := node.(type) {
		case ast.ImportGroupNode:
			for _, imp := range n.Imports {
				add(imp)
			}
		case ast.ImportNode:
			add(n)
		}
	}
	return out
}
