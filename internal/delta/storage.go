package delta

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"recengine/internal/helpers"
)

type RandomAccessFile interface {
	io.Reader
	io.Writer
	io.Closer
	io.Seeker
	Truncate(size int64) error
}

// An item of the delta list corresponding to some user.
type itemDelta struct {
	item uint64
	// OpAdd or OpRemove
	op Operation
}

// Represents a storage of the database difference data.
// The delta data complements the data stored in an associated RECDB database,
// which is immutable in its turn.
// Rougly speaking, the delta file for a database is something like a patch file
// for a Git branch.
type DeltaStorage struct {
	// The whole file data loaded into the memory.
	// The map key stores the user id.
	deltaCache map[uint64][]itemDelta
	// Unsaved changes. The map key stores the user id.
	newDelta map[uint64][]itemDelta
	// Total item count (flushed + unflushed).
	totalItemCount int
	// Number of unflushed items.
	unflushedItemCount int
	// Storage file.
	file RandomAccessFile
}

// If the file is corrupted, recovers it making its data consistent.
// All inconsistent data is skipped (removed).
// The file is considered corrupted if it's locked, which means it hasn't been
// closed properly.
func Recover(file RandomAccessFile) error {
	tmpFile := helpers.NewFileBuffer(nil)
	err := recoverTo(file, tmpFile)
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
func Open(file RandomAccessFile) (*DeltaStorage, error) {
	size, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, err
	}

	storage := DeltaStorage{
		deltaCache:         make(map[uint64][]itemDelta),
		newDelta:           make(map[uint64][]itemDelta),
		totalItemCount:     0,
		unflushedItemCount: 0,
		file:               file,
	}

	if size == 0 {
		// Create a file
		err = writePrefix(file)
		if err != nil {
			return nil, err
		}
		hdr := header{
			version:    version,
			locked:     0,
			numEntries: 0,
		}
		err = writeHeader(&hdr, file)
		if err != nil {
			return nil, err
		}
	} else {
		// Read delta cache
		_, err := file.Seek(0, io.SeekStart)
		if err != nil {
			return nil, err
		}
		err = readPrefix(file)
		if err != nil {
			return nil, err
		}
		hdr := header{}
		err = readHeader(&hdr, file)
		if err != nil {
			return nil, err
		}
		if hdr.locked != 0 {
			return nil, errors.New("The file is corrupted (locked)")
		}
		storage.totalItemCount = int(hdr.numEntries)
		entry := entry{}
		for i := 0; i < int(hdr.numEntries); i++ {
			err = readEntry(&entry, file)
			if err != nil {
				return nil, fmt.Errorf("Cannot read %dth entry: %v", i, err)
			}
			items, exists := storage.deltaCache[entry.user]
			if !exists {
				items = make([]itemDelta, 0, 100)
				storage.deltaCache[entry.user] = items
			}
			storage.deltaCache[entry.user] = append(items, itemDelta{
				op:   entry.op,
				item: entry.item,
			})
		}
	}

	// Lock the file
	err = writeLocked(true, file)
	if err != nil {
		return nil, fmt.Errorf("Failed to lock the file: %v", err)
	}

	return &storage, nil
}

// Opens a delta storage file.
// If the file is empty, writes all necessary data.
// If the file is corrupted, tries to recover it first.
func OpenMaybeRecover(file RandomAccessFile) (*DeltaStorage, error) {
	locked, err := IsLocked(file)
	if locked {
		err = Recover(file)
		if err != nil {
			return nil, fmt.Errorf("Failed to recover: %v", err)
		}
	}
	storage, err := Open(file)
	if err != nil {
		file.Close()
		return nil, err
	}
	return storage, nil
}

// Rewrites file header with actual data.
func (s *DeltaStorage) flushHeader() error {
	hdr := &header{
		version:    version,
		locked:     1,
		numEntries: uint32(s.GetTotalItemCount()),
	}
	s.file.Seek(int64(len(prefix)), io.SeekStart)
	writer := bufio.NewWriter(s.file)
	err := writeHeader(hdr, writer)
	if err != nil {
		return fmt.Errorf("Failed to write header: %v", err)
	}
	err = writer.Flush()
	if err != nil {
		return fmt.Errorf("Failed to flush buffer: %v", err)
	}
	return nil
}

// Writes entries to disk.
func (s *DeltaStorage) flushEntries() error {
	// Write to file
	s.file.Seek(0, io.SeekEnd)
	writer := bufio.NewWriter(s.file)
	for user, deltas := range s.newDelta {
		for _, delta := range deltas {
			// Write to file
			dto := &entry{
				op:   delta.op,
				user: user,
				item: delta.item,
			}
			err := writeEntry(dto, writer)
			if err != nil {
				return fmt.Errorf("Failed to write entry: %v", err)
			}
			// Add to cache
			_, exists := s.deltaCache[user]
			if !exists {
				s.deltaCache[user] = make([]itemDelta, 0)
			}
			s.deltaCache[user] = append(s.deltaCache[user], delta)
		}
	}
	// Flush the buffer
	err := writer.Flush()
	if err != nil {
		return fmt.Errorf("Failed to flush buffer: %v", err)
	}
	// Reset the unflushed data
	s.unflushedItemCount = 0
	s.newDelta = make(map[uint64][]itemDelta)
	return nil
}

// Flushes the internal buffers.
func (s *DeltaStorage) Flush() error {
	err := s.flushHeader()
	if err != nil {
		return fmt.Errorf("Failed to flush header: %v", err)
	}
	err = s.flushEntries()
	if err != nil {
		return fmt.Errorf("Failed to flush entries: %v", err)
	}
	return nil
}

// Closes the delta storage file.
// The files not closed with this function are considered broken and require recovery.
func (s *DeltaStorage) Close() error {
	err := s.Flush()
	if err != nil {
		return fmt.Errorf("Failed to flush cache: %v", err)
	}
	err = writeLocked(false, s.file)
	if err != nil {
		return fmt.Errorf("Failed to lock the file: %v", err)
	}
	err = s.file.Close()
	if err != nil {
		return fmt.Errorf("Failed to close the underlying file: %v", err)
	}
	return nil
}

// Returns the number of items currently stored in the storage.
func (s *DeltaStorage) GetUserCount() int {
	unflushedCount := 0
	for user := range s.newDelta {
		_, exists := s.deltaCache[user]
		if !exists {
			unflushedCount++
		}
	}
	return unflushedCount + len(s.deltaCache)
}

// Returns the number of items currently stored in the storage.
func (s *DeltaStorage) GetTotalItemCount() int {
	return s.totalItemCount
}

// Returns the storage file size required to keep all the data.
func (s *DeltaStorage) GetFileSize() uint64 {
	return uint64(s.GetTotalItemCount())*entrySize + headerSize + uint64(len(prefix))
}

// Returns the last operation associated with the specified user-item pair.
// This method exists mostly for debugging and testing purposes.
func (s *DeltaStorage) Get(user uint64, item uint64) (Operation, bool) {
	deltas, exists := s.newDelta[user]
	if !exists {
		deltas, exists = s.deltaCache[user]
	}
	if !exists {
		return OpAdd, false
	}
	for i := range deltas {
		delta := deltas[len(deltas)-i-1]
		if delta.item == item {
			return delta.op, true
		}
	}
	return OpAdd, false
}

// Adds an operation of item addition or removal to a user profile.
func (s *DeltaStorage) Add(op Operation, user uint64, item uint64) {
	deltas, exists := s.newDelta[user]
	if !exists {
		s.newDelta[user] = make([]itemDelta, 0)
	} else {
		// Try to replace the duplicate or opposite operation (if any)
		for i := range deltas {
			if deltas[i].item == item {
				deltas[i].op = op
				return
			}
		}
	}
	// Add a new operation
	s.newDelta[user] = append(deltas, itemDelta{
		item: item,
		op:   op,
	})
	s.unflushedItemCount++
	s.totalItemCount++
}
