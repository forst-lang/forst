package basic

import (
	"fmt"
)

type TooShort struct {
	message string
}

func (e TooShort) Error() string {
	return e.message
}

func greet(name string) (string, error) {
	if len(name) == 0 {
		return "", TooShort{"Name must be at least 1 character long"}
	}
	return fmt.Sprintf("Hello, %s!", name), nil
}
