// forst-gen-file-routes emits a sealed HTTP route registry from file-based .Router() types.
package main

import (
	"forst/internal/genplugin"
)

const version = "0.1.0"

func main() {
	genplugin.Run(emitFileRoutes)
}
