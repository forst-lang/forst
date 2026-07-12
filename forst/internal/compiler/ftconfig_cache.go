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
		// Do not pass DevSession.ParsedFiles here: entry-package parse caching runs before
		// modulecheck and would hide sibling packages (e.g. forst/bcrypt.ft) from ScanModule.
	}
	result, err := modulecheck.CheckModuleProviders(c.log, opts)
	if err == nil && result != nil && c.Args.DevSession != nil {
		c.Args.DevSession.StoreModuleResult(moduleRoot, result)
	}
	return result, err
}
