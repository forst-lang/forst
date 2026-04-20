package main

import "fmt"
import errors "errors"
import os "os"

func main() {
	catalog := map[string]int{"ITEM-1": 10}
	sku := "ITEM-1"
	avail, availErr := func() (int, error) {
		v, ok := catalog[sku]
		if !ok {
			return 0, errors.New("missing map key")
		}
		return v, nil
	}()
	if !(availErr == nil) {
		os.Exit(1)
	}
	fmt.Println(string(avail))
	fmt.Println("ok")
}
