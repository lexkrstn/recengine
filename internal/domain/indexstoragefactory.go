package domain

import "io"

// Index storage factory.
type IndexStorageFactory interface {
	// Opens an index file by the specified path. If the file doesn't exist yet
	// it will be created.
	OpenFile(filePath string) (IndexStorage, error)

	// Opens an index file by the specified path. If the file doesn't exist yet
	// it will be created.
	Open(file io.ReadWriteSeeker, closer io.Closer) (IndexStorage, error)
}
