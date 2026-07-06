// Package testutil provides shared helpers for Forst compiler pipeline tests.
//
// Layering (avoids Go import cycles):
//   - testutil: logger, module root, parse, assert, opts, mixed-module fixtures
//   - typechecker: Typecheck / MustTypecheck / MustTypecheckMerged (harness.go)
//   - transformergo: MustCompileGo / MustCompileMergedGo (harness.go)
//
// Use MustTypecheck / MustCompileGo for in-process parse → typecheck → transform
// tests. For end-to-end CLI/compiler benchmarks, keep using compiler.CompileFile
// (see internal/compiler/bench_test.go) — that path exercises module providers,
// discovery, and file I/O not covered here.
package testutil
