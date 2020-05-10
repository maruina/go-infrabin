package main

import (
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

	var resp helpers.Response
	resp.Hostname = hostname
	resp.KubeResponse = &helpers.KubeResponse{
		PodName:   helpers.GetEnv("POD_NAME", ""),
		Namespace: helpers.GetEnv("POD_NAMESPACE", ""),
		PodIP:     helpers.GetEnv("POD_IP", ""),
		NodeName:  helpers.GetEnv("NODE_NAME", ""),
	}

	data := helpers.MarshalResponseToString(resp)
	_, err = io.WriteString(w, data)
	if err != nil {
		log.Fatal("error writing to ResponseWriter: ", err)
	}
}

// LivenessHandler handles the "/liveness" endpoint
func LivenessHandler(w http.ResponseWriter, r *http.Request) {
	var resp helpers.Response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	resp.Liveness = "pass"
	data := helpers.MarshalResponseToString(resp)
	_, err := io.WriteString(w, data)
	if err != nil {
		log.Fatal("error writing to ResponseWriter", err)
	}
}

// DelayHandler handles the "/delay" endpoint
func DelayHandler(w http.ResponseWriter, r *http.Request) {
	var resp helpers.Response
	var seconds int
	vars := mux.Vars(r)
	w.Header().Set("Content-Type", "application/json")

	seconds, err := strconv.Atoi(vars["seconds"])
	if err != nil {
		resp.Error = "cannot convert seconds to integer"
		data := helpers.MarshalResponseToString(resp)
		w.WriteHeader(http.StatusBadRequest)
		_, err = io.WriteString(w, data)
		if err != nil {
			log.Fatal("error writing to ResponseWriter", err)
		}
	} else {
		maxDelay, err := strconv.Atoi(helpers.GetEnv("INFRABIN_MAX_DELAY", "120"))
		if err != nil {
			log.Fatalf("cannot convert env var INFRABIN_MAX_DELAY to integer: %v", err)
		}
		time.Sleep(time.Duration(helpers.Min(seconds, maxDelay)) * time.Second)

		resp.Delay = strconv.Itoa(seconds)
		data := helpers.MarshalResponseToString(resp)

		w.WriteHeader(http.StatusOK)
		_, err = io.WriteString(w, data)
		if err != nil {
			log.Fatal("error writing to ResponseWriter", err)
		}
	}
}

// HeadersHandler handles the headers endpoint
func HeadersHandler(w http.ResponseWriter, r *http.Request) {
	var resp helpers.Response
	w.Header().Set("Content-Type", "application/json")
	resp.Headers = &r.Header

	data := helpers.MarshalResponseToString(resp)
	_, err := io.WriteString(w, data)
	if err != nil {
		log.Fatal("error writing to ResponseWriter: ", err)
	}
}

// EnvHandler handles the env endpoint
func EnvHandler(w http.ResponseWriter, r *http.Request) {
	var resp helpers.Response
	vars := mux.Vars(r)
	w.Header().Set("Content-Type", "application/json")
	value := helpers.GetEnv(vars["env_var"], "")

	if value == "" {
		data := helpers.MarshalResponseToString(resp)
		w.WriteHeader(http.StatusNotFound)
		_, err := io.WriteString(w, data)
		if err != nil {
			log.Fatal("error writing to ResponseWriter", err)
		}
	} else {

		resp.Env = map[string]string{
			vars["env_var"]: value,
		}
		data := helpers.MarshalResponseToString(resp)

		w.WriteHeader(http.StatusOK)
		_, err := io.WriteString(w, data)
		if err != nil {
			log.Fatal("error writing to ResponseWriter", err)
		}

	}
}

func main() {
	r := mux.NewRouter()
	a := mux.NewRouter()
	finish := make(chan bool)

	r.HandleFunc("/", RootHandler)
	r.HandleFunc("/delay/{seconds}", DelayHandler)
	r.HandleFunc("/headers", HeadersHandler)
	r.HandleFunc("/env/{env_var}", EnvHandler)

	a.HandleFunc("/liveness", LivenessHandler)

	serviceSrv := &http.Server{
		Handler: r,
		Addr:    "0.0.0.0:8888",
		// Good practice: enforce timeouts
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	adminSrv := &http.Server{
		Handler: a,
		Addr:    "0.0.0.0:8899",
		// Good practice: enforce timeouts
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Print("starting go-infrabin")

	go func() {
		log.Print("Listening on service port 8888...")
		log.Fatal(serviceSrv.ListenAndServe())
	}()

	go func() {
		log.Print("Listening on admin port 8899...")
		log.Fatal(adminSrv.ListenAndServe())
	}()
	<-finish
}
