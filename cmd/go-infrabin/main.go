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
	var val string
	var seconds int
	vars := mux.Vars(r)
	w.Header().Set("Content-Type", "application/json")
	val, ok := vars["seconds"]
	if !ok {
		seconds = 0
	}

	seconds, err := strconv.Atoi(val)

	if err != nil {
		log.Printf("cannot convert vars['seconds'] to integer: %v", err)
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

func main() {
	r := mux.NewRouter()
	a := mux.NewRouter()

	r.HandleFunc("/", RootHandler)
	r.HandleFunc("/delay/{seconds}", DelayHandler)

	a.HandleFunc("/liveness", LivenessHandler)

	reqSrv := &http.Server{
		Handler: r,
		Addr:    "0.0.0.0:8888",
		// Good practice: enforce timeouts
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	adminSrv := &http.Server{
		Handler: a,
		Addr:    "0.0.0.0:8889",
		// Good practice: enforce timeouts
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	log.Print("starting go-infrabin")
	go log.Fatal(reqSrv.ListenAndServe())
	go log.Fatal(adminSrv.ListenAndServe())
	select {} // block forever
}
