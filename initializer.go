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

	// Preprocessor is called on every git request.
	Preprocessor struct {
		// Update the code now
		Process func(params *ProcessParams) error
	}
)

// IsNil returns true if a the Preprocessor struct is nil.
func (t *Preprocessor) IsNil() bool {
	if t == nil {
		return true
	}
	return false
}
