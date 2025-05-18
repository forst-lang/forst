package main

import (
	"fmt"
	"os"
)

// ValidationError represents a domain-specific validation error
type ValidationError struct {
	Type    string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// MustBeARealName validates that a name is not empty.
// Returns a ValidationError if the name is too short.
func MustBeARealName(name string) error {
	if len(name) < 1 {
		return ValidationError{
			Type:    "TooShort",
			Message: "name must be at least 1 character long",
		}
	}
	return nil
}

// MustNotExceedSpeedLimit validates that speed is within limits.
// Returns a ValidationError if the speed exceeds the limit.
func MustNotExceedSpeedLimit(speed int) error {
	if speed >= 100 {
		return ValidationError{
			Type:    "TooFast",
			Message: "speed must not exceed 100 km/h",
		}
	}
	return nil
}

// CheckConditions validates all conditions.
// Returns an error if any validation fails.
func CheckConditions() error {
	if err := MustBeARealName("John"); err != nil {
		return fmt.Errorf("name validation failed: %w", err)
	}

	speed := 80
	if err := MustNotExceedSpeedLimit(speed); err != nil {
		return fmt.Errorf("speed validation failed: %w", err)
	}

	return nil
}

// ExampleMain demonstrates how to use the validation functions
func ExampleMain() {
	if err := CheckConditions(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Conditions met, program exiting successfully")
}
