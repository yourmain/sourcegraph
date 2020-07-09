package store

import "github.com/pkg/errors"

// ErrUnknownRepository occurs when a repository does not exist.
var ErrUnknownRepository = errors.New("unknown repository")

// ErrIllegalLimit occurs when a limit is not strictly positive.
var ErrIllegalLimit = errors.New("illegal limit")
