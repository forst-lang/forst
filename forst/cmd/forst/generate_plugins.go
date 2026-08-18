package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"forst/internal/ftconfig"
	"forst/internal/semantic"

	"github.com/sirupsen/logrus"
)

const defaultPluginTimeout = 30 * time.Second

var pluginTimeout = defaultPluginTimeout

func runSemanticPlugins(boundaryRoot string, snapshot *semantic.GenerateRequest, plugins []ftconfig.GeneratePluginConfig, log *logrus.Logger, stats *generateWriteStats) error {
	if len(plugins) == 0 {
		return nil
	}
	if snapshot == nil {
		return fmt.Errorf("semantic snapshot is nil")
	}
	for _, plugin := range plugins {
		if err := runOneSemanticPlugin(boundaryRoot, snapshot, plugin, log, stats); err != nil {
			return fmt.Errorf("plugin %q: %w", plugin.Name, err)
		}
	}
	return nil
}

func runOneSemanticPlugin(boundaryRoot string, snapshot *semantic.GenerateRequest, plugin ftconfig.GeneratePluginConfig, log *logrus.Logger, stats *generateWriteStats) error {
	req := *snapshot
	req.Plugin = &semantic.PluginRef{
		Name: plugin.Name,
		Opt:  plugin.Opt,
	}
	payload, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	cmdPath := plugin.ResolveCmd(boundaryRoot)
	ctx, cancel := context.WithTimeout(context.Background(), pluginTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, cmdPath)
	cmd.Dir = boundaryRoot
	cmd.Stdin = bytes.NewReader(payload)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	log.WithFields(logrus.Fields{
		"plugin": plugin.Name,
		"cmd":    cmdPath,
		"out":    plugin.Out,
	}).Debug("Running semantic plugin")

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("timed out after %s", pluginTimeout)
		}
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return fmt.Errorf("%w: %s", err, msg)
		}
		return err
	}
	if stderr.Len() > 0 {
		log.WithFields(logrus.Fields{
			"plugin": plugin.Name,
			"stderr": strings.TrimSpace(stderr.String()),
		}).Debug("Semantic plugin stderr")
	}

	var resp semantic.GenerateResponse
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}
	if resp.ProtocolVersion != 0 && resp.ProtocolVersion != semantic.ProtocolVersion {
		return fmt.Errorf("unsupported protocolVersion %d", resp.ProtocolVersion)
	}

	outDir := plugin.EffectiveOutDir(boundaryRoot)
	for _, f := range resp.Files {
		if err := validatePluginOutputPath(f.Path); err != nil {
			return fmt.Errorf("invalid output path %q: %w", f.Path, err)
		}
		target := filepath.Join(outDir, filepath.FromSlash(f.Path))
		if err := writeGeneratedFile(target, []byte(f.Content), stats); err != nil {
			return fmt.Errorf("write %q: %w", f.Path, err)
		}
	}
	for _, d := range resp.Diagnostics {
		level := logrus.WarnLevel
		if d.Severity == "error" {
			level = logrus.ErrorLevel
		}
		log.WithFields(logrus.Fields{
			"plugin":  plugin.Name,
			"typeId":  d.TypeID,
			"message": d.Message,
		}).Log(level, "semantic plugin diagnostic")
	}
	return nil
}

func validatePluginOutputPath(path string) error {
	if path == "" {
		return fmt.Errorf("empty path")
	}
	if filepath.IsAbs(path) {
		return fmt.Errorf("absolute path not allowed")
	}
	clean := filepath.ToSlash(filepath.Clean(path))
	if clean == ".." || strings.HasPrefix(clean, "../") || strings.Contains(clean, "/../") {
		return fmt.Errorf("path escapes output directory")
	}
	if strings.Contains(clean, `\`) {
		return fmt.Errorf("backslashes not allowed; use /")
	}
	return nil
}

func dumpSemanticSnapshot(snapshot *semantic.GenerateRequest) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(snapshot)
}
