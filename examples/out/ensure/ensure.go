package main

import (
	errors "errors"
	"fmt"
	os "os"
)

// T_EzZKiw9FNu2: TypeDefShapeExpr({})
type T_EzZKiw9FNu2 struct {
}

// T_SZ37bhJn94J: TypeDefShapeExpr({})
type T_SZ37bhJn94J struct {
}

// T_Saiim6wHvrR: TypeDefShapeExpr({})
type T_Saiim6wHvrR struct {
}

// T_iw8no2aCk8H: TypeDefShapeExpr({})
type T_iw8no2aCk8H struct {
}

func checkConditions() (int, error) {
	err := mustBeARealName("John")
	if err != nil {
		return 0, errors.New("assertion failed: Error.Nil()")
	}
	speed := 80
	err = mustNotExceedSpeedLimit(speed)
	if err != nil {
		return 0, errors.New("assertion failed: Error.Nil()")
	}
	return 20, nil
}
func main() {
	result, resultErr := checkConditions()
	if !(resultErr == nil) {
		fmt.Printf("Conditions not met: %s", resultErr.Error())
		fmt.Println()
		os.Exit(1)
	}
	fmt.Printf("Conditions met (value %d), program exiting successfully", result)
	fmt.Println()
}
func mustBeARealName(name string) error {
	if len(name) < 1 {
		return errors.New("assertion failed: String.Min(1)")
	}
	return nil
}
func mustNotExceedSpeedLimit(speed int) error {
	if speed >= 100 {
		return errors.New("assertion failed: Int.LessThan(100)")
	}
	return nil
}
