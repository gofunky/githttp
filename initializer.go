package githttp

type (
	// The initializer parameters provided
	ProcessParams struct {
		// The public path to the git repository
		Repository string
		// Local path of the git repository where the files are located
		LocalPath string
		// If the repository has just been created
		IsNew bool
	}

	// The result of the preprocessor
	ProcessResult struct {
		// If the processing fails or the parameters don't meet the requirements, otherwise nil
		Err error
	}

	// To be implemented by the preprocessor
	Preprocessor struct {
		// Update the code now
		Process func(params *ProcessParams) ProcessResult
	}
)

func (t *Preprocessor) IsNil() bool {
	if t == nil {
		return true
	}
	return false
}
