package factory

import (
	"github.com/supergiant/capacity/pkg/providers"
	"github.com/supergiant/capacity/pkg/providers/aws"
)

func New(provider string, config providers.Config) (providers.Provider, error) {
	switch provider {
	case aws.Name:
		return aws.New(config)
	}
	return nil, ErrNotSuported
}
