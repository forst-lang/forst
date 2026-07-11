package main

import "fmt"
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
		os.Exit(1)
	}
}
