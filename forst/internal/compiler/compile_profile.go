package compiler

import (
	"time"

	"github.com/sirupsen/logrus"
)

// CompileSandboxTiming holds sandbox write phase durations for reload profiling.
type CompileSandboxTiming struct {
	WriteSandboxMs int64
	GoModTidyMs    int64
}

// CompilePhaseTimings holds compile sub-phase durations for reload profiling.
type CompilePhaseTimings struct {
	LoadParseMs    int64
	TypecheckMs    int64
	CodegenMs      int64
	WriteSandboxMs int64
	GoModTidyMs    int64
	GoBuildMs      int64
}

func (c *Compiler) reloadProfileEnabled() bool {
	return c != nil && (c.Args.ReloadProfile || c.Args.ReportPhases)
}

func (c *Compiler) logCompilePhaseTiming(t CompilePhaseTimings) {
	if c == nil || !c.reloadProfileEnabled() || c.log == nil {
		return
	}
	c.log.WithFields(logrus.Fields{
		"event":            "compile_timing",
		"load_parse_ms":    t.LoadParseMs,
		"typecheck_ms":     t.TypecheckMs,
		"codegen_ms":       t.CodegenMs,
		"write_sandbox_ms": t.WriteSandboxMs,
		"go_mod_tidy_ms":   t.GoModTidyMs,
		"go_build_ms":      t.GoBuildMs,
	}).Info("compile phases")
}

func (c *Compiler) takeCompileTimings() CompilePhaseTimings {
	if c == nil {
		return CompilePhaseTimings{}
	}
	t := c.lastCompileTimings
	c.lastCompileTimings = CompilePhaseTimings{}
	return t
}

// TakeCompileTimings returns compile sub-phase timings from the most recent compileToGo.
func (c *Compiler) TakeCompileTimings() CompilePhaseTimings {
	return c.takeCompileTimings()
}

// LogCompilePhaseTiming emits structured compile sub-phase timing when ReloadProfile is enabled.
func (c *Compiler) LogCompilePhaseTiming(t CompilePhaseTimings) {
	c.logCompilePhaseTiming(t)
}

func elapsedMs(since time.Time) int64 {
	return time.Since(since).Milliseconds()
}
