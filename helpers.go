package main

import (
	"encoding/json"
	"net/http"
)

// jsonResponse writes a map[string]any to the ResponseWriter by encoding the body as JSON
func jsonResponse(w http.ResponseWriter, body map[string]any, err error) {
	data := map[string]any{
		"body":   body,
		"status": "ok",
	}

	// Write headers and response based on if an error occurred.
	if err == nil {
		w.WriteHeader(http.StatusOK)
	} else {
		data["status"] = "error"
		data["error"] = err.Error()
		w.WriteHeader(http.StatusBadRequest)
	}

	// Set the Content-Type header
	w.Header().Set("Content-Type", "application/json")

	// Encode the JSON response to the ResponseWriter
	enc := json.NewEncoder(w)
	err = enc.Encode(data)
	if err != nil {
		panic(err)
	}
}
