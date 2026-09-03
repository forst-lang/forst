package main

import "fmt"
import os "os"
// PebbleFail: TypeDefErrorExpr({reason: String})
type PebbleFail struct {
	reason string
}
// T_CQ83zP8NNan: TypeDefShapeExpr({})
type T_CQ83zP8NNan struct {
}

func (e PebbleFail) Error() string {
	return "error"
}
func (e PebbleFail) ForstErrorTag() string {
	return "main/PebbleFail"
}
func checkFlag(ok bool) (string, error) {
	if !ok {
		return "", PebbleFail{reason: "flag was false"}
	}
	return "ok", nil
}
func main() {
	r, rErr := checkFlag(true)
	if !(rErr == nil) {
		{
			fmt.Fprintf(os.Stderr, "ensure failed: %v\n", rErr)
			os.Exit(1)
		}
	}
	fmt.Println(r)
}
