package factory

import (
	"strings"

	"github.com/supergiant/capacity/pkg/provider"
	"github.com/supergiant/capacity/pkg/provider/aws"
)

// New returns a new provider and acts as a switch for selecting a
// specific provider type. For example, sending "aws" as the provider
// will return an AWS provider for the Capacity Controller to use.
func New(clusterName, provider string, config provider.Config) (provider.Provider, error) {
	clusterName = strings.TrimSpace(clusterName)
	if clusterName == "" {
		return nil, ErrNoClusterName
	}

	switch provider {
	case aws.Name:
		return aws.New(clusterName, config)
	}
	return nil, ErrNotSuported
}
