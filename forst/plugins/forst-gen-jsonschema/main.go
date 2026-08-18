// forst-gen-jsonschema emits JSON Schema from a Forst semantic snapshot.
package main

import (
	"forst/internal/genplugin"
)

const version = "0.1.0"

func main() {
	genplugin.Run(emitJSONSchema)
}
