package githttp

import (
	"errors"
	"fmt"
)

// MissingArgument is to be returned if there are git server options missing that are passed to the factory.
var MissingArgument = errors.New("Insufficient factory options options provided.")

// ErrorNoAccess is a error with the path to the requested repository
type ErrorNoAccess struct {
	// Path to directory of repo accessed
	Dir string
}

func (e *ErrorNoAccess) Error() string {
	return fmt.Sprintf("Could not access repo at '%s'", e.Dir)
}
