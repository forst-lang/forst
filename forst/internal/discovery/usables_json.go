package discovery

// UsablesDiscoveryV1 is the Go-side discovery JSON schema (SPEC § Discovery JSON v1).
type UsablesDiscoveryV1 struct {
	Version  int                            `json:"version"`
	Packages map[string]UsablesPackageV1    `json:"packages"`
}

// UsablesPackageV1 holds per-function Usables metadata for one package path.
type UsablesPackageV1 struct {
	Functions map[string]UsablesFunctionV1 `json:"functions"`
}

// UsablesFunctionV1 is the v1 per-function Usables summary.
type UsablesFunctionV1 struct {
	Usables  []string `json:"usables"`
	Runnable bool     `json:"runnable"`
}

// BuildUsablesDiscoveryV1 builds schema v1 from discovered FunctionInfo maps.
func BuildUsablesDiscoveryV1(functions map[string]map[string]FunctionInfo) UsablesDiscoveryV1 {
	out := UsablesDiscoveryV1{
		Version:  1,
		Packages: make(map[string]UsablesPackageV1),
	}
	for pkg, fns := range functions {
		pkgEntry := UsablesPackageV1{Functions: make(map[string]UsablesFunctionV1)}
		for name, info := range fns {
			pkgEntry.Functions[name] = UsablesFunctionV1{
				Usables:  append([]string(nil), info.Usables...),
				Runnable: info.Runnable,
			}
		}
		out.Packages[pkg] = pkgEntry
	}
	return out
}
