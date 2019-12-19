package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	helpers "github.com/maruina/go-infrabin/internal/helpers"
)

// RootHandler handles the "/" endpoint
func RootHandler(w http.ResponseWriter, r *http.Request) {
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalf("cannot get hostname: %v", err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	podName := helpers.GetEnv("POD_NAME", "null")
	namespace := helpers.GetEnv("POD_NAMESPACE", "null")
	podIP := helpers.GetEnv("POD_ID", "null")
	nodeName := helpers.GetEnv("NODE_NAME", "null")

	jsonData := `
	{"hostname": "%s"},
	{"kubernetes": {
		"pod": "%s",
		"namespace": "%s",
		"ip": "%s",
		"node": "%s"
	}}
	`

	resp := fmt.Sprintf(`

	`, hostname, podName, namespace, podIP, nodeName)
	_, err = io.WriteString(w, resp)
	if err != nil {
		log.Fatal("error writing to ResponseWriter", err)
	}
}

// LivenessHandler handles the "/healthcheck/liveness" endpoint
func LivenessHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err := io.WriteString(w, `{"status": "liveness probe healthy"}`)
	if err != nil {
		log.Fatal("error writing to ResponseWriter", err)
	}
}

// DelayHandler handles the "/delay" endpoint
func DelayHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	seconds, err := strconv.Atoi(vars["seconds"])
	if err != nil {
		log.Fatalf("cannot convert vars['seconds'] to integer: %v", err)
	}
	maxDelay, err := strconv.Atoi(helpers.GetEnv("INFRABIN_MAX_DELAY", "120"))
	if err != nil {
		log.Fatalf("cannot convert env var INFRABIN_MAX_DELAY to integer: %v", err)
	}
	time.Sleep(time.Duration(helpers.Min(seconds, maxDelay)) * time.Second)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	resp := fmt.Sprintf(`{"status": "completed", "delay": "%d"}`, seconds)
	_, err = io.WriteString(w, resp)
	if err != nil {
		log.Fatal("error writing to ResponseWriter", err)
	}
}

func main() {
	r := mux.NewRouter()

	r.HandleFunc("/", RootHandler)
	r.HandleFunc("/delay/{seconds}", DelayHandler)
	r.HandleFunc("/healthcheck/liveness", LivenessHandler)

	srv := &http.Server{
		Handler: r,
		Addr:    "0.0.0.0:8080",
		// Good practice: enforce timeouts
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	log.Print("starting go-infrabin")
	log.Fatal(srv.ListenAndServe())
}
