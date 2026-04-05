// Package coveragehotspots documents Tier-1 test coverage hotspots for the Forst compiler.
//
// Use this as a prioritization guide—not a line-coverage target. Re-run periodically:
//
//	go test -coverprofile=c.out ./internal/typechecker/... ./internal/transformer/go/...
//	go tool cover -func=c.out | sort -t$'\t' -k3 -n | head -40
//
// Cross-check with git churn (recent edits) on the same paths.
//
// # Shortlist (high blast radius)
//
//   - internal/typechecker: typechecker.go, infer.go, collect.go, register.go, unify_shape.go, unify_is.go, narrowing
//     (see unify_is_test.go for `is` + type guards through CheckTypes)
//   - internal/transformer/go: statement.go, expression.go, shape.go, ensure*.go, typeguard.go
//   - internal/goload, cmd/forst/lsp: merged-package analysis and cross-file behavior
//   - internal/parser: utils.go (parse error strings), internal/forstpkg: merge + ParseForstFile
//
// Tier 3 (supporting): internal/printer ops.go and typeprint.go — prefer stem-matched ops_test.go /
// typeprint_test.go with small golden assertions on operator and type printing.
//
// # Validation / codegen emit tests
//
// Pipeline tests that assert generated Go for constraints, type guards, and ensure blocks use the
// TestEmitValidation_ name prefix (see internal/transformer/go/pipeline_integration_test.go). That
// keeps “validation emit” coverage grep-friendly and distinct from generic statement coverage.
package coveragehotspots
