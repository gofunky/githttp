package githttp

import (
	gogit "gopkg.in/src-d/go-git.v4"
)

type (
	// ProcessParams contain the preprocessing parameters.
	ProcessParams struct {
		// The public path to the git repository
		RepositoryPath string
		// Local path of the git repository where the files are located
		LocalPath string
		// If the repository has just been created
		IsNew bool
		// The gogit repository
		Repository *gogit.Repository
	}

	// Preprocesser is called on every git request.
	Preprocesser struct {
		// Process updates the code.
		Process func(params *ProcessParams) error
		// Path checks if the requested uri is valid and returns a deterministic local repository path.
		Path func(rawPath string) (targetPath string, err error)
	}
)

// IsProcessNil returns true if a the Preprocesser struct or the Process func is nil.
func (t *Preprocesser) IsProcessNil() bool {
	if t == nil || t.Process == nil {
		return true
	}
	return false
}

// IsPathNil returns true if a the Preprocesser struct or the IsPathNil func is nil.
func (t *Preprocesser) IsPathNil() bool {
	if t == nil || t.Path == nil {
		return true
	}
	return false
}
