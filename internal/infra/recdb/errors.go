package recdb

type LockedFileError struct{}

func (e *LockedFileError) Error() string {
	return "The file hasn't been closed property, it may be corrupted"
}

func NewLockedFileError() error {
	return &LockedFileError{}
}
