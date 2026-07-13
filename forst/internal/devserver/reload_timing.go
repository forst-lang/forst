package devserver

import (
	"os"
	"time"

	"forst/internal/ftconfig"
	"forst/internal/modulecheck"

	"github.com/sirupsen/logrus"
)

const envReloadTrace = "FORST_RELOAD_TRACE"

type reloadTiming struct {
	start           time.Time
	markReloadingMs int64
	forstCompileMs  int64
	goBuildMs       int64
	childStopMs     int64
	portPickMs      int64
	goStartMs       int64
	invokeReadyMs   int64
}

func reloadProfileEnabled(cfg *ftconfig.Config) bool {
	if os.Getenv(envReloadTrace) == "1" {
		return true
	}
	return cfg != nil && (cfg.Dev.Watch || cfg.Dev.HotReload)
}

func newReloadTiming() *reloadTiming {
	return &reloadTiming{start: time.Now()}
}

func (rt *reloadTiming) recordMarkReloading(since time.Time) {
	rt.markReloadingMs = time.Since(since).Milliseconds()
}

func (rt *reloadTiming) recordForstCompile(since time.Time, goBuildMs int64) {
	rt.forstCompileMs = time.Since(since).Milliseconds()
	rt.goBuildMs = goBuildMs
}

func (rt *reloadTiming) recordChildStop(since time.Time) {
	rt.childStopMs = time.Since(since).Milliseconds()
}

func (rt *reloadTiming) recordPortPick(since time.Time) {
	rt.portPickMs = time.Since(since).Milliseconds()
}

func (rt *reloadTiming) recordGoStart(since time.Time) {
	rt.goStartMs = time.Since(since).Milliseconds()
}

func (rt *reloadTiming) recordInvokeReady(since time.Time) {
	rt.invokeReadyMs = time.Since(since).Milliseconds()
}

func (rt *reloadTiming) logComplete(log *logrus.Logger, gen uint64, success bool) {
	if log == nil || rt == nil {
		return
	}
	fields := logrus.Fields{
		"event":              "reload_timing",
		"generation":         gen,
		"success":            success,
		"mark_reloading_ms":  rt.markReloadingMs,
		"forst_compile_ms":   rt.forstCompileMs,
		"go_build_ms":        rt.goBuildMs,
		"child_stop_ms":      rt.childStopMs,
		"port_pick_ms":       rt.portPickMs,
		"go_start_ms":        rt.goStartMs,
		"invoke_ready_ms":    rt.invokeReadyMs,
		"modulecheck_passes": modulecheck.PassCount(),
		"total_ms":           time.Since(rt.start).Milliseconds(),
	}
	log.WithFields(fields).Info("reload complete")
}

func (rt *reloadTiming) logStart(log *logrus.Logger, gen uint64) {
	if log == nil || rt == nil {
		return
	}
	log.WithFields(logrus.Fields{
		"event":      "reload_timing_start",
		"generation": gen,
	}).Info("reload started")
}
