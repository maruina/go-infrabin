package helpers

import (
	"fmt"
	"log"
	"testing"
)

func TestMarshalStructToString(t *testing.T) {
	var resp Response
	resp.Hostname = "hal"

	data := MarshalStructToString(resp)
	expected := fmt.Sprint(`{"hostname":"hal"}`)
	if data != expected {
		log.Fatalf("error: expected %v got %v", expected, data)
	}
}
