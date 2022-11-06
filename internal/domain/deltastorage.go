package domain

// OpAdd or OpRemove
type DeltaOp byte

const (
	DeltaOpAdd    DeltaOp = '+'
	DeltaOpRemove DeltaOp = '-'
)

// Represents a storage of the database difference data.
// The delta data complements the data stored in an associated RECDB database,
// which is immutable in its turn.
// Rougly speaking, the delta file for a database is something like a patch file
// for a Git branch.
type DeltaStorage interface {
	// Flushes the internal buffers.
	Flush() error

	// Closes the delta storage file.  The files not closed with this
	// function are considered broken and require recovery.
	Close() error

	// Returns the number of users currently stored in the storage.
	GetUserCount() int

	// Returns the number of items currently stored in the storage.
	GetTotalItemCount() int

	// Returns the storage file size required to keep all the data.
	GetFileSize() uint64

	// Returns the last operation associated with the specified user-item pair.
	// This method exists mostly for debugging and testing purposes.
	Get(user uint64, item uint64) (DeltaOp, bool)

	// Adds an operation of item addition or removal to a user profile.
	Add(op DeltaOp, user uint64, item uint64)
}
