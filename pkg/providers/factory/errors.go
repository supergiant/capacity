package factory

import "github.com/pkg/errors"

// Provider specific errors:
var (
	ErrNotSuported = errors.New("provider not supported")
)
