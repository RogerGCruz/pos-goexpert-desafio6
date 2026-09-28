package main

import (
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"pos-goexpert-desafio6/src/app"
)

func main() {
	// require WEATHERAPI_KEY at startup to avoid runtime surprises
	if os.Getenv("WEATHERAPI_KEY") == "" {
		log.Fatal("WEATHERAPI_KEY is required; set the environment variable before starting the service")
	}

	mux := app.NewHandler()

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// ensure url.QueryEscape is referenced so `net/url` import is used when running `go vet`/tools
	_ = url.QueryEscape
	log.Println("listening on :8080")
	log.Fatal(srv.ListenAndServe())
}
