package main

import "regexp"
import fmt "fmt"
import os "os"
// RedactOut: TypeDefShapeExpr({Text: String})
type RedactOut struct {
	Text string
}

func Redact(text string) (RedactOut, error) {
	re := regexp.MustCompile("a+")
	out := text
	out = re.ReplaceAllStringFunc(out, func(m string) string {
		return "x"
	})
	return RedactOut{Text: out}, nil
}
func main() {
	r, rErr := Redact("baa")
	if !(rErr == nil) {
		{
			fmt.Fprintf(os.Stderr, "ensure failed: %v\n", rErr)
			os.Exit(1)
		}
	}
	fmt.Println(r.Text)
}
