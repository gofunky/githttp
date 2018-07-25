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

	// ProcessResult contains the result of the preprocessor.
	ProcessResult struct {
		// If the processing fails or the parameters don't meet the requirements, otherwise nil
		Err error
	}

	// Preprocessor is called on every git request.
	Preprocessor struct {
		// Update the code now
		Process func(params *ProcessParams) ProcessResult
	}
)

// IsNil returns true if a the Preprocessor struct is nil.
func (t *Preprocessor) IsNil() bool {
	if t == nil {
		return true
	}
	return false
}
