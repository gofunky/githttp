package githttp

import (
	"errors"
	"fmt"
)

// ErrMissingArgument is to be returned if there are git server options missing that are passed to the factory.
var ErrMissingArgument = errors.New("insufficient factory options options provided")

// ErrorNoAccess is a error with the path to the requested repository
type ErrorNoAccess struct {
	// Path to directory of repo accessed
	Dir string
}

func (e *ErrorNoAccess) Error() string {
	return fmt.Sprintf("could not access repo at '%s'", e.Dir)
}
