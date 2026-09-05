package main

import "fmt"
import "net/http"

func main() {
	mux := http.NewServeMux()
	srv := &http.Server{}
	srv.Handler = mux
	fmt.Println(srv.Handler != nil)
}
