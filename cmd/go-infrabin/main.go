package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
)

// RootHandler handles the "/" path
func RootHandler(w http.ResponseWriter, r *http.Request) {
	name, err := os.Hostname()
	if err != nil {
		log.Fatalf("cannot get hostname: %v", err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	resp := fmt.Sprintf(`{"status": "running", "hostname": "%s"}`, name)
	io.WriteString(w, resp)
}

// LivenessHandler handles the "/healthcheck/liveness" path
func LivenessHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, `{"status": "liveness probe healthy"}`)
}

func main() {
	r := mux.NewRouter()

	r.HandleFunc("/", RootHandler)
	r.HandleFunc("/healthcheck/liveness", LivenessHandler)

	srv := &http.Server{
		Handler: r,
		Addr:    "127.0.0.1:8080",
		// Good practice: enforce timeouts
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	log.Print("starting go-infrabin")
	log.Fatal(srv.ListenAndServe())
}
