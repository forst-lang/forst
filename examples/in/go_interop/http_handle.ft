package main

import "net/http"

func registerRoutes() {
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		println("ok")
	})
}
