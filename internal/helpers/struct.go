package helpers

import (
	"encoding/json"
	"log"
	"net/http"
)

// Response creates the go-infrabin main response
type Response struct {
	Hostname     string        `json:"hostname,omitempty"`
	KubeResponse *KubeResponse `json:"kubernetes,omitempty"`
	Liveness     string        `json:"liveness,omitempty"`
	Delay        string        `json:"delay,omitempty"`
	Error        string        `json:"error,omitempty"`
	Headers      *http.Header  `json:"headers,omitempty"`
}

// KubeResponse creates the response if running on Kubernetes
type KubeResponse struct {
	PodName   string `json:"pod_name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	PodIP     string `json:"pod_ip,omitempty"`
	NodeName  string `json:"node_name,omitempty"`
}

// MarshalResponseToString marhal a Response struct into a json and return it as string
func MarshalResponseToString(r Response) string {
	data, err := json.Marshal(r)
	if err != nil {
		log.Fatal("error marshal object: ", err)
	}
	return string(data)
}
