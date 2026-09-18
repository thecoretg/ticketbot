package handlers

import (
	"net/http"
)

func HandleHealthCheck(w http.ResponseWriter, r *http.Request) {
	res := struct {
		Result string `json:"result"`
	}{Result: "success"}

	outputJSON(w, res)
}
