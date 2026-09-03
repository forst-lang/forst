package main

import "regexp"
import fmt "fmt"
import os "os"
// WillowOut: TypeDefShapeExpr({Text: String})
type WillowOut struct {
	Text string
}

func RedactWillow(text string) (WillowOut, error) {
	re := regexp.MustCompile("a+")
	out := text
	out = re.ReplaceAllStringFunc(out, func(m string) string {
		return "x"
	})
	return WillowOut{Text: out}, nil
}
func main() {
	r, rErr := RedactWillow("baa")
	if !(rErr == nil) {
		{
			fmt.Fprintf(os.Stderr, "ensure failed: %v\n", rErr)
			os.Exit(1)
		}
	}
	fmt.Println(r.Text)
}
