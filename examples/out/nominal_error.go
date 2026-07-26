package main

import "fmt"
import errors "errors"
import os "os"
// NotPositive: TypeDefErrorExpr({message: String})
type NotPositive struct {
	message string
}
// T_H4c2uQ34ZJV: TypeDefShapeExpr({})
type T_H4c2uQ34ZJV struct {
}
// T_iw8no2aCk8H: TypeDefShapeExpr({})
type T_iw8no2aCk8H struct {
}

func (e NotPositive) Error() string {
	return "error"
}
func Test() error {
	n := 0
	if n <= 0 {
		return NotPositive{message: "n must be greater than 0"}
	}
	return nil
}
func main() {
	err := Test()
	if err != nil {
		fmt.Println(err)
		{
			fmt.Fprintf(os.Stderr, "ensure failed: %v\n", errors.New("ensure err is Error.Nil(): want nil"))
			os.Exit(1)
		}
	}
}
