package factory

import "github.com/pkg/errors"

// Provider specific errors:
var (
	ErrNoClusterName = errors.New("cluster name should be provided")
	ErrNotSuported   = errors.New("provider not supported")
)
