package goload

import "golang.org/x/tools/go/packages"

// PackagesLoader loads Go packages (typically packages.Load).
type PackagesLoader func(cfg *packages.Config, patterns ...string) ([]*packages.Package, error)

type loadConfig struct {
	loader PackagesLoader
}

// LoadOpt configures a LoadByPkgPath call.
type LoadOpt func(*loadConfig)

// WithPackagesLoader overrides the package loader for one call (tests inject fakes).
func WithPackagesLoader(l PackagesLoader) LoadOpt {
	return func(c *loadConfig) {
		c.loader = l
	}
}

func resolveLoadConfig(opts []LoadOpt) loadConfig {
	cfg := loadConfig{loader: packagesLoadFn}
	for _, o := range opts {
		if o != nil {
			o(&cfg)
		}
	}
	if cfg.loader == nil {
		cfg.loader = packages.Load
	}
	return cfg
}

// SetPackagesLoaderForTest replaces the process-wide loader; returns restore func.
func SetPackagesLoaderForTest(l PackagesLoader) func() {
	prev := packagesLoadFn
	if l == nil {
		packagesLoadFn = packages.Load
	} else {
		packagesLoadFn = l
	}
	return func() { packagesLoadFn = prev }
}
