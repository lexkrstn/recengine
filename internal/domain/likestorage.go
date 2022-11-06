package domain

// Represents an abstract like profiile data storage.
type LikeStorage interface {
	// Closes the storage file. The files not closed with this
	// function are considered broken and require recovery.
	Close() error

	// Executes a set of tasks sequently reading and/or modifiying entries in
	// the corresponding database file.
	ProcessActions(actions []Action) error
}
