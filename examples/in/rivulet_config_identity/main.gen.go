package main

import (
	"example.com/rivulet_config_identity/app"
	"example.com/rivulet_config_identity/config"
)
import errors "errors"
import fmt "fmt"
import os "os"
// T_iw8no2aCk8H: TypeDefShapeExpr({})
type T_iw8no2aCk8H struct {
}

func main() {
	cfg := config.Load()
	err := app.Run(cfg)
	if err != nil {
		{
			fmt.Fprintf(os.Stderr, "ensure failed: %v\n", errors.New("ensure err is Error.Nil(): want nil"))
			os.Exit(1)
		}
	}
	fmt.Println(cfg.Port)
}
