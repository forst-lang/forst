package main

import (
	"fmt"
	"os"
)

type TooShort struct {
	message string
}

func (e TooShort) Error() string {
	return e.message
}

type TooFast struct {
	message string
}

func (e TooFast) Error() string {
	return e.message
}

func MustBeARealName(name string) error {
	if len(name) < 1 {
		return TooShort{"Name must be at least 1 character long"}
	}
	return nil
}

func MustNotExceedSpeedLimit(speed int) error {
	if speed >= 100 {
		return TooFast{"Speed must not exceed 100 km/h"}
	}
	return nil
}

func CheckConditions() error {
	if err := MustBeARealName("John"); err != nil {
		return err
	}
	if err := MustNotExceedSpeedLimit(101); err != nil {
		return err
	}
	return nil
}

func Main() {
	if err := CheckConditions(); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
