package infrabin

import (
	"log"
	"net/http"
	"os"
	"io"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/maruina/go-infrabin/pkg/helpers"
)

// RootHandler handles the "/" endpoint
func RootHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	fail := helpers.GetEnv("FAIL_ROOT_HANDLER", "")
	if fail != "" {
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)

		hostname, err := os.Hostname()
		if err != nil {
			log.Fatalf("cannot get hostname: %v", err)
		}

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
