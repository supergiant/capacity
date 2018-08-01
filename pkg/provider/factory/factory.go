package factory

import (
	"strings"

	"github.com/supergiant/capacity/pkg/provider"
	"github.com/supergiant/capacity/pkg/provider/aws"
)

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
