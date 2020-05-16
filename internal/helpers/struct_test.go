package helpers

import (
	"log"
	"testing"
)

func TestMarshalResponseToString(t *testing.T) {
	var resp Response
	resp.Hostname = "hal"

	data := MarshalResponseToString(resp)
	expected := `{"hostname":"hal"}`
	if data != expected {
		log.Fatalf("error: expected %v got %v", expected, data)
	}
}
