package githttp

type (
	// ProcessParams contain the preprocessing parameters.
	ProcessParams struct {
		// The public path to the git repository
		Repository string
		// Local path of the git repository where the files are located
		LocalPath string
		// If the repository has just been created
		IsNew bool
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
