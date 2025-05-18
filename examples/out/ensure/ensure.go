package main

import (
	errors "errors"
	"fmt"
	"os"
)

func mustBeARealName(name string) error {
	if len(name) < 1 {
		return errors.New("TooShort([\"Name must be at least 1 character long\"])")
	}
	return nil
}
func mustNotExceedSpeedLimit(speed int) error {
	if speed >= 100 {
		return errors.New("TooFast([\"Speed must not exceed 100 km/h\"])")
	}
	return nil
}
func checkConditions() error {
	err := mustBeARealName("John")
	if err != nil {
		return err
	}
	speed := 80
	err = mustNotExceedSpeedLimit(speed)
	if err != nil {
		return err
	}
	return nil
}
func main() {
	err := checkConditions()
	if err != nil {
		fmt.Printf("Conditions not met: %s", err.Error())
		fmt.Println()
		os.Exit(1)
		panic(errors.New("assertion failed: " + "TYPE_ERROR.Nil()"))
	}
	fmt.Println("Conditions met, program exiting successfully")
}
