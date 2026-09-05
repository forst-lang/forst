package semantic

import "encoding/json"

const ProtocolVersion = 1

// GenerateRequest is the semantic snapshot sent to plugins on stdin.
type GenerateRequest struct {
	ProtocolVersion int              `json:"protocolVersion"`
	CompilerVersion string           `json:"compilerVersion"`
	Plugin          *PluginRef       `json:"plugin,omitempty"`
	Module          ModuleInfo         `json:"module"`
	Packages        []SemanticPackage  `json:"packages"`
	Types           map[string]Type    `json:"types"`
	Functions       map[string]Function `json:"functions"`
}

// GenerateResponse is read from plugin stdout.
type GenerateResponse struct {
	ProtocolVersion int          `json:"protocolVersion"`
	Files           []OutputFile `json:"files"`
	Diagnostics     []Diagnostic `json:"diagnostics,omitempty"`
}

type PluginRef struct {
	Name string          `json:"name"`
	Opt  json.RawMessage `json:"opt,omitempty"`
}

type ModuleInfo struct {
	GoModule string `json:"goModule"`
	Root     string `json:"root"`
}

type SemanticPackage struct {
	Name        string   `json:"name"`
	Dir         string   `json:"dir"`
	Files       []string `json:"files"`
	TypeIDs     []string `json:"typeIds"`
	FunctionIDs []string `json:"functionIds"`
}

type SourceSpan struct {
	File      string `json:"file"`
	StartLine int    `json:"startLine,omitempty"`
	StartCol  int    `json:"startCol,omitempty"`
	EndLine   int    `json:"endLine,omitempty"`
	EndCol    int    `json:"endCol,omitempty"`
}

type Constraint struct {
	Name    string        `json:"name"`
	Args    []any         `json:"args,omitempty"`
	Origin  string        `json:"origin"`
	Applies string        `json:"applies,omitempty"`
}

type Type struct {
	ID           string       `json:"id"`
	Kind         string       `json:"kind"`
	Constraints  []Constraint `json:"constraints,omitempty"`
	AliasedTo    string       `json:"aliasedTo,omitempty"`
	Underlying   string       `json:"underlying,omitempty"`
	Element      string       `json:"element,omitempty"`
	Key          string       `json:"key,omitempty"`
	Value        string       `json:"value,omitempty"`
	Inner        string       `json:"inner,omitempty"`
	Length       *int         `json:"length,omitempty"`
	Members      []string     `json:"members,omitempty"`
	Success      string       `json:"success,omitempty"`
	Failure      string       `json:"failure,omitempty"`
	Params       []FuncParam  `json:"params,omitempty"`
	Returns      []string     `json:"returns,omitempty"`
	Fields       []ShapeField `json:"fields,omitempty"`
	Payload      string       `json:"payload,omitempty"`
	ImportPath   string       `json:"importPath,omitempty"`
	GoName       string       `json:"name,omitempty"`
	Visibility   string       `json:"visibility,omitempty"`
	Doc          string       `json:"doc,omitempty"`
	Span         *SourceSpan  `json:"span,omitempty"`
	Debug        string       `json:"debug,omitempty"`
}

type ShapeField struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Optional bool   `json:"optional,omitempty"`
	Embedded bool   `json:"embedded,omitempty"`
	Tag      string `json:"tag,omitempty"`
	Method   bool   `json:"method,omitempty"`
	Function string `json:"function,omitempty"`
}

type FuncParam struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Variadic bool   `json:"variadic,omitempty"`
}

type Function struct {
	ID              string       `json:"id"`
	Name            string       `json:"name"`
	Package         string       `json:"package"`
	Visibility      string       `json:"visibility"`
	Role            string       `json:"role"`
	Runnable        bool         `json:"runnable"`
	Receiver        *string      `json:"receiver"`
	Params          []FuncParam  `json:"params"`
	Input           string       `json:"input"`
	Returns         []string     `json:"returns"`
	ErrorSet        ErrorSet     `json:"errorSet"`
	Providers       []string     `json:"providers"`
	Doc             string       `json:"doc,omitempty"`
	Span            *SourceSpan  `json:"span,omitempty"`
}

type ErrorSet struct {
	Nominal         []string `json:"nominal,omitempty"`
	UnknownPossible bool     `json:"unknownPossible"`
}

type OutputFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type Diagnostic struct {
	Severity string      `json:"severity"`
	Message  string      `json:"message"`
	TypeID   string      `json:"typeId,omitempty"`
	Span     *SourceSpan `json:"span,omitempty"`
}
