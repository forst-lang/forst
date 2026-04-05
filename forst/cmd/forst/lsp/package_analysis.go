package lsp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"forst/internal/ast"
	"forst/internal/lexer"
	"forst/internal/parser"
	"forst/internal/typechecker"

	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

// packageMergeInfo is set on forstDocumentContext when the typechecker analyzed a merged
// same-package group (multiple open .ft buffers in one directory).
type packageMergeInfo struct {
	MemberURIs  []string
	TokensByURI map[string][]ast.Token
}

// fileParseResult holds lex/parse output for one URI in a package group.
type fileParseResult struct {
	URI      string
	FilePath string
	Content  string
	FileID   string
	Tokens   []ast.Token
	Nodes    []ast.Node
	ParseErr error
}

// packageSnapshot is a cached merged analysis for one fingerprint of open buffers.
type packageSnapshot struct {
	uris        []string
	results     []fileParseResult
	mergedNodes []ast.Node
	tc          *typechecker.TypeChecker
	checkErr    error
}

// samePackageOpenURIs returns sorted file:// URIs for open .ft buffers in the same directory
// as anchorURI with the same Forst package clause. The anchor is included when it matches.
func (s *LSPServer) samePackageOpenURIs(anchorURI string) []string {
	if !isForstDocumentURI(anchorURI) {
		return []string{anchorURI}
	}
	anchorPath := filePathFromDocumentURI(anchorURI)
	dir := filepath.Dir(anchorPath)

	s.documentMu.RLock()
	anchorContent := s.openDocuments[anchorURI]
	s.documentMu.RUnlock()
	if anchorContent == "" {
		b, err := os.ReadFile(anchorPath)
		if err != nil {
			return []string{anchorURI}
		}
		anchorContent = string(b)
	}
	pkg := forstPackageNameFromContent(anchorContent)
	if pkg == "" {
		return []string{anchorURI}
	}

	var out []string
	s.documentMu.RLock()
	for u, text := range s.openDocuments {
		if u == anchorURI {
			continue
		}
		if !isForstDocumentURI(u) {
			continue
		}
		p := filePathFromDocumentURI(u)
		if filepath.Dir(p) != dir {
			continue
		}
		if forstPackageNameFromContent(text) != pkg {
			continue
		}
		out = append(out, u)
	}
	s.documentMu.RUnlock()

	out = append(out, anchorURI)
	sort.Slice(out, func(i, j int) bool {
		return filePathFromDocumentURI(out[i]) < filePathFromDocumentURI(out[j])
	})
	return out
}

// loadPackageGroupContents returns the text used for lex/parse and fingerprinting: open buffer
// first, then disk for any missing buffer.
func (s *LSPServer) loadPackageGroupContents(uris []string) (map[string]string, error) {
	s.documentMu.RLock()
	contents := make(map[string]string, len(uris))
	for _, u := range uris {
		contents[u] = s.openDocuments[u]
	}
	s.documentMu.RUnlock()

	for _, u := range uris {
		if contents[u] != "" {
			continue
		}
		p := filePathFromDocumentURI(u)
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		contents[u] = string(b)
	}
	return contents, nil
}

func packageGroupFingerprintFromContents(uris []string, contents map[string]string) string {
	var b strings.Builder
	for _, u := range uris {
		b.WriteString(u)
		b.WriteByte(0)
		b.WriteString(contents[u])
		b.WriteByte(0)
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}

func parseLimit(n int) int {
	if n < 1 {
		return 1
	}
	if n > 4 {
		return 4
	}
	return n
}

// parsePackageGroupMembersParallel lexes and parses each URI; results[i] corresponds to uris[i].
func (s *LSPServer) parsePackageGroupMembersParallel(uris []string, contents map[string]string) ([]fileParseResult, error) {
	cd, ok := s.debugger.(*CompilerDebugger)
	if !ok {
		return nil, fmt.Errorf("compiler debugger unavailable")
	}
	packageStore := cd.packageStore

	results := make([]fileParseResult, len(uris))
	g, _ := errgroup.WithContext(context.Background())
	g.SetLimit(parseLimit(len(uris)))

	for i := range uris {
		i := i
		g.Go(func() error {
			u := uris[i]
			fp := filePathFromDocumentURI(u)
			pkgPath := extractPackagePath(fp)
			fid := string(packageStore.RegisterFile(fp, pkgPath))
			content := contents[u]

			lex := lexer.New([]byte(content), fid, s.log)
			tokens := lex.Lex()
			psr := parser.New(tokens, fid, s.log)
			var nodes []ast.Node
			var parseErr error
			func() {
				defer func() {
					if r := recover(); r != nil {
						if pe, ok := r.(*parser.ParseError); ok {
							parseErr = pe
						} else {
							parseErr = fmt.Errorf("parser panic: %v", r)
						}
					}
				}()
				nodes, parseErr = psr.ParseFile()
			}()

			results[i] = fileParseResult{
				URI:      u,
				FilePath: fp,
				Content:  content,
				FileID:   fid,
				Tokens:   tokens,
				Nodes:    nodes,
				ParseErr: parseErr,
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}
	return results, nil
}

func mergePackageNodes(results []fileParseResult) []ast.Node {
	var merged []ast.Node
	for i := range results {
		if results[i].ParseErr != nil {
			continue
		}
		merged = append(merged, results[i].Nodes...)
	}
	return merged
}

func (s *LSPServer) buildPackageSnapshot(uris []string, results []fileParseResult, workDir string) *packageSnapshot {
	merged := mergePackageNodes(results)
	if len(merged) == 0 {
		return &packageSnapshot{uris: uris, results: results}
	}
	tc := typechecker.New(s.log, false)
	tc.GoWorkspaceDir = workDir
	checkErr := tc.CheckTypes(merged)
	return &packageSnapshot{
		uris:        uris,
		results:     results,
		mergedNodes: merged,
		tc:          tc,
		checkErr:    checkErr,
	}
}

func (s *LSPServer) snapshotToDocumentContext(snap *packageSnapshot, uri string) *forstDocumentContext {
	tokensByURI := make(map[string][]ast.Token, len(snap.results))
	var local *fileParseResult
	for i := range snap.results {
		r := &snap.results[i]
		tokensByURI[r.URI] = r.Tokens
		if r.URI == uri {
			cp := *r
			local = &cp
		}
	}
	if local == nil {
		return nil
	}

	merge := &packageMergeInfo{
		MemberURIs:  append([]string(nil), snap.uris...),
		TokensByURI: tokensByURI,
	}

	return &forstDocumentContext{
		URI:          local.URI,
		FilePath:     local.FilePath,
		Content:      local.Content,
		FileID:       local.FileID,
		Tokens:       local.Tokens,
		Nodes:        local.Nodes,
		ParseErr:     local.ParseErr,
		TC:           snap.tc,
		CheckErr:     snap.checkErr,
		PackageMerge: merge,
	}
}

// analyzePackageGroupMerged returns a merged snapshot when the open package group fully parses.
// memberURIs may be nil to compute membership from the anchor (same as samePackageOpenURIs).
func (s *LSPServer) analyzePackageGroupMerged(anchorURI string, memberURIs []string) (snap *packageSnapshot, ctx *forstDocumentContext, ok bool) {
	defer func() {
		if r := recover(); r != nil {
			s.log.WithFields(logrus.Fields{
				"function": "analyzePackageGroupMerged",
				"panic":    r,
			}).Debug("analyzePackageGroupMerged panic recovered")
			snap = nil
			ctx = nil
			ok = false
		}
	}()

	uris := memberURIs
	if uris == nil {
		uris = s.samePackageOpenURIs(anchorURI)
	} else {
		found := false
		for _, u := range uris {
			if u == anchorURI {
				found = true
				break
			}
		}
		if !found {
			uris = s.samePackageOpenURIs(anchorURI)
		}
	}
	if len(uris) <= 1 {
		return nil, nil, false
	}

	contents, err := s.loadPackageGroupContents(uris)
	if err != nil {
		return nil, nil, false
	}
	fp := packageGroupFingerprintFromContents(uris, contents)

	if cached := s.packageAnalysis.get(fp); cached != nil && cached.tc != nil {
		if ctx := s.snapshotToDocumentContext(cached, anchorURI); ctx != nil {
			return cached, ctx, true
		}
	}

	results, err := s.parsePackageGroupMembersParallel(uris, contents)
	if err != nil {
		return nil, nil, false
	}
	for i := range results {
		if results[i].ParseErr != nil {
			return nil, nil, false
		}
	}

	workDir := filepath.Dir(filePathFromDocumentURI(uris[0]))
	snap = s.buildPackageSnapshot(uris, results, workDir)
	s.packageAnalysis.put(fp, snap)

	return snap, s.snapshotToDocumentContext(snap, anchorURI), true
}

func (s *LSPServer) invalidatePackageAnalysisCache() {
	s.packageAnalysis.clear()
}
