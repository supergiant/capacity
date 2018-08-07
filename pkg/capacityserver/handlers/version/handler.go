package version

import (
	"encoding/json"
	"net/http"

	"github.com/supergiant/capacity/pkg/log"
	"github.com/supergiant/capacity/pkg/version"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	// swagger:route GET /version version getVersion
	//
	// Returns version information.
	//
	// This will show version information.
	//
	//     Produces:
	//     - application/json
	//
	//     Schemes: https, http
	//
	//     Responses:
	//     200: versionResponse

	if err := json.NewEncoder(w).Encode(version.Get()); err != nil {
		log.Errorf("handler: version: failed to write response: %v", err)
	}
}
