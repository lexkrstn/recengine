package domain

// LikeStorage factory.
type LikeStorageFactory interface {
	// If the file is corrupted, recovers it making its data consistent.
	// All inconsistent data is skipped (removed).  The file is considered
	// corrupted if it's locked, which means it hasn't been closed properly.
	Recover(file RandomAccessFile) error

	// Opens a storage file. If the file is empty, writes all necessary data.
	// Like storage also depends on a corresponding delta and index storage
	// objects, but it doesn't close them automatically upon closing itself.
	Open(
		file RandomAccessFile,
		deltaStorage DeltaStorage,
		indexStorage IndexStorage,
	) (LikeStorage, error)

	// Opens a storage file.  If the file is empty, writes all necessary
	// data. If the file is corrupted, tries to recover it first.
	// Like storage also depends on a corresponding delta and index storage
	// objects, but it doesn't close them automatically upon closing itself.
	OpenMaybeRecover(
		file RandomAccessFile,
		deltaStorage DeltaStorage,
		indexStorage IndexStorage,
	) (LikeStorage, error)
}
