package delta

import (
	"errors"
	"fmt"
	"io"
	"recengine/internal/domain"
	"recengine/internal/helpers"
)

// Delta storage factory.
type storageFactory struct {
	proto Protocol
}

// Compile-time type check
var _ = (domain.DeltaStorageFactory)((*storageFactory)(nil))

// Instantiates a delta storage factory.
func NewStorageFactory() domain.DeltaStorageFactory {
	return NewStorageFactoryForProtocol(NewProtocol())
}

// Instantiates a delta storage factory.
func NewStorageFactoryForProtocol(proto Protocol) domain.DeltaStorageFactory {
	return &storageFactory{
		proto: proto,
	}
}

// If the file is corrupted, recovers it making its data consistdomain.
// All inconsistent data is skipped (removed).  The file is considered
// corrupted if it's locked, which means it hasn't been closed properly.
func (f *storageFactory) Recover(file domain.RandomAccessFile) error {
	tmpFile := helpers.NewFileBuffer(nil)
	err := f.proto.RecoverTo(file, tmpFile)
	if err != nil {
		return err
	}
	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}
	_, err = tmpFile.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}
	var size int64
	size, err = io.Copy(file, tmpFile)
	if err != nil {
		return err
	}
	err = file.Truncate(size)
	if err != nil {
		return err
	}
	_, err = file.Seek(0, io.SeekStart)
	return err
}

// Opens a delta storage file. If the file is empty, writes all necessary data.
func (f *storageFactory) Open(file domain.RandomAccessFile) (domain.DeltaStorage, error) {
	size, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, err
	}

	storage := storage{
		deltaCache:         make(map[uint64][]itemDelta),
		newDelta:           make(map[uint64][]itemDelta),
		totalItemCount:     0,
		unflushedItemCount: 0,
		file:               file,
		proto:              f.proto,
	}

	if size == 0 {
		// Create a file
		err = f.proto.WritePrefix(file)
		if err != nil {
			return nil, err
		}
		hdr := Header{
			Version:    Version,
			Locked:     0,
			NumEntries: 0,
		}
		err = f.proto.WriteHeader(&hdr, file)
		if err != nil {
			return nil, err
		}
	} else {
		// Read delta cache
		_, err := file.Seek(0, io.SeekStart)
		if err != nil {
			return nil, err
		}
		err = f.proto.ReadPrefix(file)
		if err != nil {
			return nil, err
		}
		hdr := Header{}
		err = f.proto.ReadHeader(&hdr, file)
		if err != nil {
			return nil, err
		}
		if hdr.Locked != 0 {
			return nil, errors.New("the file is corrupted (locked)")
		}
		storage.totalItemCount = int(hdr.NumEntries)
		entry := Entry{}
		for i := 0; i < int(hdr.NumEntries); i++ {
			err = f.proto.ReadEntry(&entry, file)
			if err != nil {
				return nil, fmt.Errorf("cannot read %dth entry: %v", i, err)
			}
			items, exists := storage.deltaCache[entry.UserID]
			if !exists {
				items = make([]itemDelta, 0, 100)
				storage.deltaCache[entry.UserID] = items
			}
			storage.deltaCache[entry.UserID] = append(items, itemDelta{
				op:   entry.Op,
				item: entry.ItemID,
			})
		}
	}

	// Lock the file
	err = f.proto.WriteLocked(true, file)
	if err != nil {
		return nil, fmt.Errorf("failed to lock the file: %v", err)
	}

	return &storage, nil
}

// Opens a delta storage file.  If the file is empty, writes all necessary
// data. If the file is corrupted, tries to recover it first.
func (f *storageFactory) OpenMaybeRecover(file domain.RandomAccessFile) (domain.DeltaStorage, error) {
	locked, err := f.proto.IsLocked(file)
	if err != nil {
		return nil, fmt.Errorf("failed to check if file is locked: %v", err)
	}
	if locked {
		err = f.Recover(file)
		if err != nil {
			return nil, fmt.Errorf("failed to recover: %v", err)
		}
	}
	storage, err := f.Open(file)
	if err != nil {
		file.Close()
		return nil, err
	}
	return storage, nil
}
