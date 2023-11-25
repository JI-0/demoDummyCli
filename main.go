package main

import (
	"crypto/tls"
	"log"
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
		TLSConfig: &tls.Config{
			GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
				cert, err := tls.LoadX509KeyPair("/etc/letsencrypt/live/the.testingwebrtc.com/fullchain.pem",
					"/etc/letsencrypt/live/the.testingwebrtc.com/privkey.pem")
				if err != nil {
					log.Println("Failed to load TLS certificate!")
					return nil, err
				}
				return &cert, nil
			},
		},
	}

	srv.ListenAndServeTLS("", "")

	println("Shutting down dummy cli server")
}
