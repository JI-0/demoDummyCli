package main

import (
	"fmt"
	"net/http"
)

func main() {
	println("Starting dummy cli server")
	manager := NewManager()

	mux := http.NewServeMux()
	mux.HandleFunc("/", manager.serveWS)

	srv := &http.Server{
		Addr:    ":3002",
		Handler: mux,
	}

	fmt.Print(srv.ListenAndServe())

	println("Shutting down dummy cli server")
}
