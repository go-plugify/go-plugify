package goplugify

var (
	ErrInvalidLoaderSource = NewError("invalid loader source")
	ErrPluginNoLoadMethod = NewError("plugin has no load method")
)

func NewError(message string) error {
	return &PlugifyError{message: message}
}

type PlugifyError struct {
	message string
}

func (e *PlugifyError) Error() string {
	return e.message
}