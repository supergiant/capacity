package handlers

import "net/http"

func getStatus(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("not implemented\n"))
}
