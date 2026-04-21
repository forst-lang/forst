package main

import "fmt"
import errors "errors"
import os "os"

var errMissingMapKey = errors.New("missing map key")

func main() {
	catalog := map[string]int{"ITEM-1": 10}
	sku := "ITEM-1"
	avail, availErr := func() (int, error) {
		v, ok := catalog[sku]
		if !ok {
			return 0, errMissingMapKey
		}
		return v, nil
	}()
	if !(availErr == nil) {
		os.Exit(1)
	}
	fmt.Println(string(avail))
	fmt.Println("ok")
}
