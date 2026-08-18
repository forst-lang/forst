// forst-gen-orpc emits an oRPC/tRPC-shaped TypeScript contract from marked router types.
package main

import (
	"forst/internal/genplugin"
)

const version = "0.1.0"

func main() {
	genplugin.Run(emitORPC)
}
