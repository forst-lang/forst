package forstpkg

import (
	"fmt"

	"forst/internal/parser"
)

func parseRecoverToError(displayPath string, r any) error {
	if pe, ok := r.(*parser.ParseError); ok {
		return pe
	}
	return fmt.Errorf("parse panic in %s: %v", displayPath, r)
}
