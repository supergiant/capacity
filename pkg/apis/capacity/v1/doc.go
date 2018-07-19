// Api versions allow the api contract for a resource to be changed while keeping
// backward compatibility by support multiple concurrent versions
// of the same resource

// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=package,register
// +k8s:conversion-gen=github.com/supergiant/capacity/pkg/apis/capacity
// +k8s:defaulter-gen=TypeMeta
// +groupName=capacity.supergiant.io
package v1 // import "github.com/supergiant/capacity/pkg/apis/capacity/v1"
