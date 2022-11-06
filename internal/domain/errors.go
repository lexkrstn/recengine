package domain

type CorruptedFileError struct{}

func (e *CorruptedFileError) Error() string {
	return "The file hasn't been closed property, it may be corrupted"
}

func NewCorruptedFileError() error {
	return &CorruptedFileError{}
}
