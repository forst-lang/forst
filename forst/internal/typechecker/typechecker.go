package typechecker

import (
	"forst/internal/ast"
	"forst/internal/hasher"
	"go/types"

	"github.com/sirupsen/logrus"
)

// compoundNarrowingInfo holds predicate metadata for dotted identifiers (e.g. req.state after ensure).
type compoundNarrowingInfo struct {
	guards []string
	disp   string
}

// TypeChecker performs type inference and type checking on the AST
type TypeChecker struct {
	// Maps structural hashes of AST nodes to their inferred or declared types
	Types map[NodeHash][]ast.TypeNode
	// Maps type identifiers to their definition nodes
	Defs map[ast.TypeIdent]ast.Node
	// Maps type identifiers to nodes where they are referenced
	Uses map[ast.TypeIdent][]ast.Node
	// Maps function identifiers to their parameter and return type signatures
	Functions  map[ast.Identifier]FunctionSignature
	Hasher     *hasher.StructuralHasher
	path       NodePath // Tracks current position while traversing AST
	scopeStack *ScopeStack
	// Map of inferred types for nodes
	InferredTypes map[NodeHash][]ast.TypeNode
	// Map of inferred variable types
	VariableTypes map[ast.Identifier][]ast.TypeNode
	// Per-occurrence inferred types when the parser set Ident.Span (hover / narrowing).
	variableOccurrenceTypes map[variableOccurrenceKey][]ast.TypeNode
	// Per-occurrence type guard names from if-branch / ensure narrowing (when InferAssertionType
	// preserves a named alias without Assertion on the TypeNode).
	variableOccurrenceNarrowingGuards map[variableOccurrenceKey][]string
	// Per-occurrence dotted predicate display from narrowing RHS (e.g. `MyStr().Min(12)`), for LSP hover.
	variableOccurrenceNarrowingPredicateDisplay map[variableOccurrenceKey]string
	// compoundNarrowingByIdentifier stores narrowing for dotted ensure/if subjects (e.g. req.state) by
	// identifier only. Per-span maps miss when the same path appears later with a different span.
	compoundNarrowingByIdentifier map[ast.Identifier]compoundNarrowingInfo
	// Map of inferred function return types
	FunctionReturnTypes map[ast.Identifier][]ast.TypeNode
	// List of imported packages
	imports []ast.ImportNode
	// GoWorkspaceDir is the directory used as go/packages Config.Dir for Forst <-> Go boundary checks (optional).
	GoWorkspaceDir string
	// goPkgsByLocal maps Forst import local name (e.g. fmt, bar) to loaded *types.Package for Forst <-> Go boundary checks (optional).
	goPkgsByLocal map[string]*types.Package
	// importPathByLocal maps import local identifier -> Go import path (for hover even when go/packages failed).
	importPathByLocal map[string]string
	// Logger for the type checker
	log *logrus.Logger
	// Whether to report phases
	reportPhases bool
	// loopDepth counts nested for-loop bodies for break/continue validation
	loopDepth int
	// ifChainNarrowingStack records per-if-chain narrowing events (`x is …`) for merge/join (narrow_if.go).
	ifChainNarrowingStack [][]narrowingEvent
	// currentFunction is set while inferring a function body (Ok/Err need Result(S,F) from the signature).
	currentFunction *ast.FunctionNode
	// resultErrIfBranchDepth counts nested `if subject is Err(...)` then-bodies (built-in Result
	// narrowing). Failure propagation with `return Err(...)` there is rejected; use `ensure` instead.
	resultErrIfBranchDepth int
	// shapeExpectations maps a missing named parameter type (not in Defs) to the shape inferred for
	// a shape literal that was checked with that contextual type. Enables IsTypeCompatible(hash, T)
	// when T was intentionally absent from Defs but inferShapeType still produced a concrete shape.
	shapeExpectations map[ast.TypeIdent]ast.ShapeNode
}

// New creates a new TypeChecker
func New(log *logrus.Logger, reportPhases bool) *TypeChecker {
	if log == nil {
		log = logrus.New()
		log.Warnf("No logger provided, using default logger")
	}
	h := hasher.New()
	tc := &TypeChecker{
		Types:                             make(map[NodeHash][]ast.TypeNode),
		Defs:                              make(map[ast.TypeIdent]ast.Node),
		Uses:                              make(map[ast.TypeIdent][]ast.Node),
		Functions:                         make(map[ast.Identifier]FunctionSignature),
		Hasher:                            h,
		path:                              make(NodePath, 0),
		scopeStack:                        NewScopeStack(h, log),
		InferredTypes:                     make(map[NodeHash][]ast.TypeNode),
		VariableTypes:                     make(map[ast.Identifier][]ast.TypeNode),
		variableOccurrenceTypes:           make(map[variableOccurrenceKey][]ast.TypeNode),
		variableOccurrenceNarrowingGuards: make(map[variableOccurrenceKey][]string),
		variableOccurrenceNarrowingPredicateDisplay: make(map[variableOccurrenceKey]string),
		compoundNarrowingByIdentifier:               make(map[ast.Identifier]compoundNarrowingInfo),
		FunctionReturnTypes:                         make(map[ast.Identifier][]ast.TypeNode),
		log:                                         log,
		reportPhases:                                reportPhases,
	}

	return tc
}

// GoImportPackageLoaded reports whether go/packages successfully loaded the given import local name
// (e.g. "strconv", "fmt") for Forst↔Go boundary typing. When false, qualified calls may fall back to builtins only.
func (tc *TypeChecker) GoImportPackageLoaded(local string) bool {
	return tc.goPkgsByLocal != nil && tc.goPkgsByLocal[local] != nil
}

// CheckTypes performs type inference in two passes:
// 1. Collects explicit type declarations and function signatures
// 2. Infers types for expressions and statements
func (tc *TypeChecker) CheckTypes(nodes []ast.Node) error {
	if tc.reportPhases {
		tc.log.WithFields(logrus.Fields{
			"function": "CheckTypes",
		}).Info("First pass: collecting explicit types and function signatures")
	}

	// Collect order is independent of source order: types and type guards before functions so
	// signatures can reference any user-defined type in the program (multi-file or single file).
	collectOrder := partitionTopLevelForCollect(nodes)
	for _, node := range collectOrder {
		tc.path = append(tc.path, node)
		if err := tc.collectExplicitTypes(node); err != nil {
			return err
		}
		tc.path = tc.path[:len(tc.path)-1]
	}

	tc.initGoImportPackages()

	if err := tc.validateReferencedTypesAfterCollect(); err != nil {
		return err
	}

	tc.log.WithFields(logrus.Fields{
		"imports":   len(tc.imports),
		"typeDefs":  len(tc.Defs),
		"functions": len(tc.Functions),
		"uses":      len(tc.Uses),
		"function":  "CheckTypes",
	}).Debug("Collected types and function signatures")

	if tc.reportPhases {
		tc.log.WithFields(logrus.Fields{
			"function": "CheckTypes",
		}).Info("Starting second pass: inferring types")
	}

	for _, node := range nodes {
		tc.path = append(tc.path, node)
		if _, err := tc.inferNodeType(node); err != nil {
			return err
		}
		tc.path = tc.path[:len(tc.path)-1]
	}

	return nil
}
