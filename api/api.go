package handler

import (
	"encoding/json"
	"log"
	"net/http"
)

func writeErr(w http.ResponseWriter, _ *http.Request, code int, err error) {
	var errBody struct {
		Error string `json:"error"`
	}

	if err != nil {
		errBody.Error = err.Error()
		log.Println("request", code, "error:", err)
	} else {
		errBody.Error = http.StatusText(code)
		log.Println("request", code, "error")
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(errBody)
}
