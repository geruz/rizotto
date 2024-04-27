package err

type WrappedError struct {
	Message string
	Err     error
}

func newWrappedError(message string, err error) *WrappedError {
	return &WrappedError{
		Message: message,
		Err:     err,
	}
}

func (ve *WrappedError) Error() string {
	return ve.Message + ": " + ve.Err.Error()
}

func (ve *WrappedError) Unwrap() error {
	return ve.Err
}

func Wrap(message string, err error) error {
	if err == nil {
		return nil
	}

	return newWrappedError(message, err)
}
