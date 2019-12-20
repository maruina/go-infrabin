package main

import (
	"encoding/json"
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

// Response creates the go-infrabin main response
type Response struct {
	Hostname     string        `json:"hostname"`
	KubeResponse *KubeResponse `json:"kubernetes"`
}

// KubeResponse creates the response if running on Kubernetes
type KubeResponse struct {
	PodName   string `json:"pod_name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	PodIP     string `json:"pod_ip,omitempty"`
	NodeName  string `json:"node_name,omitempty"`
}

// RootHandler handles the "/" endpoint
func RootHandler(w http.ResponseWriter, r *http.Request) {
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalf("cannot get hostname: %v", err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	var resp Response
	resp.Hostname = hostname
	resp.KubeResponse = &KubeResponse{
		PodName:   helpers.GetEnv("POD_NAME", ""),
		Namespace: helpers.GetEnv("POD_NAMESPACE", ""),
		PodIP:     helpers.GetEnv("POD_IP", ""),
		NodeName:  helpers.GetEnv("NODE_NAME", ""),
	}

	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Fatal("error marshal object: ", err)
	}
	_, err = io.WriteString(w, string(jsonResp))
	if err != nil {
		log.Fatal("error writing to ResponseWriter: ", err)
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
