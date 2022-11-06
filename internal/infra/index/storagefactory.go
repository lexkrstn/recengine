package index

import (
	"fmt"
	"io"
	"os"
	"recengine/internal/domain"
)

// Index storage factory.
type storageFactory struct {
	proto Protocol
}

// Compile-time type check
var _ = (domain.IndexStorageFactory)((*storageFactory)(nil))

// Instantiates an index storage factory.
func NewStorageFactory() domain.IndexStorageFactory {
	return NewStorageFactoryForProtocol(NewProtocol())
}

// Instantiates an index storage factory.
func NewStorageFactoryForProtocol(proto Protocol) domain.IndexStorageFactory {
	return &storageFactory{
		proto: proto,
	}
}

// Opens an index file by the specified path. If the file doesn't exist yet
// it will be created.
func (f *storageFactory) OpenFile(filePath string) (domain.IndexStorage, error) {
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	store, err := f.Open(file, file)
	return store, err
}

// Opens an index file by the specified path. If the file doesn't exist yet
// it will be created.
func (f *storageFactory) Open(
	file io.ReadWriteSeeker,
	closer io.Closer,
) (domain.IndexStorage, error) {
	storage := &storage{
		file:    file,
		closer:  closer,
		indices: make(map[uint64]uint64),
		proto:   f.proto,
	}
	size, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		if closer != nil {
			closer.Close()
		}
		return nil, err
	}
	if size > 0 {
		err = storage.load()
		if err != nil {
			if closer != nil {
				closer.Close()
			}
			return nil, fmt.Errorf("failed to load index: %v", err)
		}
	} else {
		err = storage.create()
		if err != nil {
			if closer != nil {
				closer.Close()
			}
			return nil, fmt.Errorf("failed to load index: %v", err)
		}
	}
	return storage, nil
}
