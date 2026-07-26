package errorscompat

import (
	"fmt"

	pkgerrors "github.com/pkg/errors"
	"github.com/cockroachdb/errors"
	"go.uber.org/multierr"
)

func WrappedPkgError(msg string) error {
	return pkgerrors.Wrap(fmt.Errorf("root: %s", msg), "wrap")
}

func CockroachdbWrapped(msg string) error {
	return errors.Wrap(fmt.Errorf("inner: %s", msg), "outer")
}

func MultierrCombined(a, b error) error {
	return multierr.Combine(a, b)
}
