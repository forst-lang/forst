package basic_class

import (
	"fmt"
)

// StringValidator is a generic interface for string validation
type StringValidator interface {
    Validate(value string) error
}

// MinLengthValidator ensures the string meets the minimum length
type MinLengthValidator struct {
    Min int
}

func (v MinLengthValidator) Validate(value string) error {
    if len(value) < v.Min {
        return fmt.Errorf("string must be at least %d characters long", v.Min)
    }
    return nil
}

// MaxLengthValidator ensures the string does not exceed the maximum length
type MaxLengthValidator struct {
    Max int
}

func (v MaxLengthValidator) Validate(value string) error {
    if len(value) > v.Max {
        return fmt.Errorf("string must be no more than %d characters long", v.Max)
    }
    return nil
}

// ValidationSet holds a list of validators for a string
type ValidationSet struct {
    validators []StringValidator
}

// Add appends a validator to the validation set
func (vs *ValidationSet) Add(validator StringValidator) {
    vs.validators = append(vs.validators, validator)
}

// Validate runs all validators against the given value
func (vs *ValidationSet) Validate(value string) error {
    for _, validator := range vs.validators {
        if err := validator.Validate(value); err != nil {
            return err
        }
    }
    return nil
}