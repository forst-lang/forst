package compiler

import (
	"fmt"

	"forst/internal/ftconfig"
	"forst/internal/modulecheck"
)

func (c *Compiler) loadFtconfig() (*ftconfig.Config, error) {
	if c == nil {
		return nil, fmt.Errorf("nil compiler")
	}
	c.ftconfigOnce.Do(func() {
		root := RunBoundaryRoot(c.Args)
		if root == "" {
			return
		}
		c.ftconfigCache, c.ftconfigErr = ftconfig.LoadFromDir(root)
	})
	return c.ftconfigCache, c.ftconfigErr
}

func (c *Compiler) checkModuleProvidersWithSession(moduleRoot string, opts modulecheck.Options) (*modulecheck.ModuleResult, error) {
	if c.Args.DevSession != nil {
		if cached, ok := c.Args.DevSession.CachedModuleResult(moduleRoot); ok {
			return cached, nil
		}
		if opts.ParsedFiles == nil {
			if parsed, err := c.Args.DevSession.ParsedFilesForModuleCheck(c.log, moduleRoot); err != nil {
				return nil, err
			} else if len(parsed) > 0 {
				opts.ParsedFiles = parsed
			}
		}
	}
	result, err := modulecheck.CheckModuleProviders(c.log, opts)
	if err == nil && result != nil && c.Args.DevSession != nil {
		c.Args.DevSession.StoreModuleResult(moduleRoot, result)
	}
	return result, err
}
