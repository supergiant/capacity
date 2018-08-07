package swagger

import "github.com/supergiant/capacity/pkg/version"

// versionResponse contains an application config parameters.
// swagger:response versionResponse
type versionResponse struct {
	// in:body
	Version *version.Info `json:"version"`
}
