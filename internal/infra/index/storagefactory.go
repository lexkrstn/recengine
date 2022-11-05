package index

import (
	"fmt"
	"io"
	"os"
)

// Index storage factory.
type IStorageFactory interface {
	OpenFile(filePath string) (*Storage, error)
	Open(file io.ReadWriteSeeker, closer io.Closer) (*Storage, error)
}

// Index storage factory.
type StorageFactory struct {
	IStorageFactory
	proto IProtocol
}

// Compile-type type check
var _ = (IStorageFactory)((*StorageFactory)(nil))

// Instantiates an index storage factory.
func NewFactory() IStorageFactory {
	return NewFactoryForProtocol(&Protocol{})
}

// Instantiates an index storage factory.
func NewFactoryForProtocol(proto IProtocol) IStorageFactory {
	return &StorageFactory{
		proto: proto,
	}
}

// Opens an index file by the specified path. If the file doesn't exist yet
// it will be created.
func (f *StorageFactory) OpenFile(filePath string) (*Storage, error) {
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	store, err := f.Open(file, file)
	return store, err
}

// Opens an index file by the specified path. If the file doesn't exist yet
// it will be created.
func (f *StorageFactory) Open(file io.ReadWriteSeeker, closer io.Closer) (*Storage, error) {
	storage := &Storage{
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
