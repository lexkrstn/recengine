package domain

// Delta storage factory.
type DeltaStorageFactory interface {
	// If the file is corrupted, recovers it making its data consistent.
	// All inconsistent data is skipped (removed).  The file is considered
	// corrupted if it's locked, which means it hasn't been closed properly.
	Recover(file RandomAccessFile) error

	// Opens a delta storage file. If the file is empty, writes all necessary data.
	Open(file RandomAccessFile) (DeltaStorage, error)

	// Opens a delta storage file.  If the file is empty, writes all necessary
	// data. If the file is corrupted, tries to recover it first.
	OpenMaybeRecover(file RandomAccessFile) (DeltaStorage, error)
}
