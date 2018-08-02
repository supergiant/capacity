package status

import (
	"net/http"
	"fmt"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "not implemented")
}
