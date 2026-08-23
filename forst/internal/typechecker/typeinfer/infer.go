package typeinfer

import "fmt"

// InferFromParams runs infer for each formal parameter index in order.
func InferFromParams(paramCount int, infer func(paramIdx int) error) error {
	for i := 0; i < paramCount; i++ {
		if err := infer(i); err != nil {
			return err
		}
	}
	return nil
}

// RequireAllBound fails when any type argument slot is still unbound.
func RequireAllBound(count int, isBound func(i int) bool, name func(i int) string) error {
	for i := 0; i < count; i++ {
		if !isBound(i) {
			return fmt.Errorf("could not infer type argument %s", name(i))
		}
	}
	return nil
}
