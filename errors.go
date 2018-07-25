package githttp

import (
	"fmt"
)

// ErrorNoAccess is a error with the path to the requested repository
type ErrorNoAccess struct {
	// Path to directory of repo accessed
	Dir string
}

func (e *ErrorNoAccess) Error() string {
	return fmt.Sprintf("Could not access repo at '%s'", e.Dir)
}
