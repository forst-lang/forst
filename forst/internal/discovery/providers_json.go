package discovery

// ProvidersDiscoveryV1 is the Go-side discovery JSON schema (SPEC § Discovery JSON v1).
type ProvidersDiscoveryV1 struct {
	Version  int                            `json:"version"`
	Packages map[string]ProvidersPackageV1    `json:"packages"`
}

// ProvidersPackageV1 holds per-function Providers metadata for one package path.
type ProvidersPackageV1 struct {
	Functions map[string]ProvidersFunctionV1 `json:"functions"`
}

// ProvidersFunctionV1 is the v1 per-function Providers summary.
type ProvidersFunctionV1 struct {
	Providers  []string `json:"providers"`
	Runnable bool     `json:"runnable"`
}

// BuildProvidersDiscoveryV1 builds schema v1 from discovered FunctionInfo maps.
func BuildProvidersDiscoveryV1(functions map[string]map[string]FunctionInfo) ProvidersDiscoveryV1 {
	out := ProvidersDiscoveryV1{
		Version:  1,
		Packages: make(map[string]ProvidersPackageV1),
	}
	for pkg, fns := range functions {
		pkgEntry := ProvidersPackageV1{Functions: make(map[string]ProvidersFunctionV1)}
		for name, info := range fns {
			pkgEntry.Functions[name] = ProvidersFunctionV1{
				Providers:  append([]string(nil), info.Providers...),
				Runnable: info.Runnable,
			}
		}
		out.Packages[pkg] = pkgEntry
	}
	return out
}
