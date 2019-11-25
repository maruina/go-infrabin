package main

import (
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// LivenessHandler handles the liveness check for the application
func LivenessHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, `{"status": "liveness probe healthy"}`)

}

func main() {
	r := mux.NewRouter()

	r.HandleFunc("/healthcheck/liveness", LivenessHandler)

	srv := &http.Server{
		Handler: r,
		Addr:    "127.0.0.1:8080",
		// Good practice: enforce timeouts
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	log.Println("Starting go-infrabin")
	log.Fatal(srv.ListenAndServe())
}
