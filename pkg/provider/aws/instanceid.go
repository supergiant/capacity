package aws

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// In case of any issues review:
//
// - k8s.io/kubernetes/pkg/cloudprovider/providers/aws/instances.go

// awsInstanceRegMatch represents Regex Match for AWS instance.
var awsInstanceRegMatch = regexp.MustCompile("^i-[^/]*$")

// GetMachineID extracts the awsInstanceID from the providerID
//
// providerID represents the id for an instance in the kubernetes API;
// the following form
//  * aws:///<zone>/<awsInstanceId>
//  * aws:////<awsInstanceId>
//  * <awsInstanceId>
//
func (p *Provider) GetMachineID(providerID string) (string, error) {
	if !strings.HasPrefix(providerID, "aws://") {
		// Assume a bare aws volume id (vol-1234...)
		// Build a URL with an empty host (AZ)
		providerID = "aws://" + "/" + "/" + providerID
	}
	url, err := url.Parse(providerID)
	if err != nil {
		return "", fmt.Errorf("invalid instance name (%s): %v", providerID, err)
	}
	if url.Scheme != "aws" {
		return "", fmt.Errorf("invalid scheme for AWS instance (%s)", providerID)
	}

	awsID := ""
	tokens := strings.Split(strings.Trim(url.Path, "/"), "/")
	if len(tokens) == 1 {
		// instanceId
		awsID = tokens[0]
	} else if len(tokens) == 2 {
		// az/instanceId
		awsID = tokens[1]
	}

	// We sanity check the resulting volume; the two known formats are
	// i-12345678 and i-12345678abcdef01
	if awsID == "" || !awsInstanceRegMatch.MatchString(awsID) {
		return "", fmt.Errorf("invalid format for AWS instance (%s)", providerID)
	}

	return awsID, nil
}
