package domain

// Database index storage.
type IndexStorage interface {
	// Closes the storage file.
	Close() error

	// Returns the index associated with the specified ID.
	Get(id uint64) (uint64, bool)

	// Associates an index with an ID.
	Put(id uint64, index uint64) error

	// Removes an index from the database.
	Remove(id uint64) error
}
