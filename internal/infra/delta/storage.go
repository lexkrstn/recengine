package delta

import (
	"bufio"
	"fmt"
	"io"
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
type IStorage interface {
	Flush() error
	Close() error
	GetUserCount() int
	GetTotalItemCount() int
	GetFileSize() uint64
	Get(user uint64, item uint64) (Operation, bool)
	Add(op Operation, user uint64, item uint64)
}

// Implements a storage of the database difference data.
type storage struct {
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
	// Delta file functions.
	proto IProtocol
}

// Compile-type type check
var _ = (IStorage)((*storage)(nil))

// Rewrites file header with actual data.
func (s *storage) flushHeader() error {
	fmt.Println(s)
	hdr := &Header{
		Version:    Version,
		Locked:     1,
		NumEntries: uint32(s.GetTotalItemCount()),
	}
	s.file.Seek(int64(len(prefix)), io.SeekStart)
	writer := bufio.NewWriter(s.file)
	err := s.proto.WriteHeader(hdr, writer)
	if err != nil {
		return fmt.Errorf("failed to write header: %v", err)
	}
	err = writer.Flush()
	if err != nil {
		return fmt.Errorf("failed to flush buffer: %v", err)
	}
	return nil
}

// Writes entries to disk.
func (s *storage) flushEntries() error {
	// Write to file
	s.file.Seek(0, io.SeekEnd)
	writer := bufio.NewWriter(s.file)
	for user, deltas := range s.newDelta {
		for _, delta := range deltas {
			// Write to file
			dto := &Entry{
				Op:     delta.op,
				UserID: user,
				ItemID: delta.item,
			}
			err := s.proto.WriteEntry(dto, writer)
			if err != nil {
				return fmt.Errorf("failed to write entry: %v", err)
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
		return fmt.Errorf("failed to flush buffer: %v", err)
	}
	// Reset the unflushed data
	s.unflushedItemCount = 0
	s.newDelta = make(map[uint64][]itemDelta)
	return nil
}

// Flushes the internal buffers.
func (s *storage) Flush() error {
	err := s.flushHeader()
	if err != nil {
		return fmt.Errorf("failed to flush header: %v", err)
	}
	err = s.flushEntries()
	if err != nil {
		return fmt.Errorf("failed to flush entries: %v", err)
	}
	return nil
}

// Closes the delta storage file.
// The files not closed with this function are considered broken and require recovery.
func (s *storage) Close() error {
	err := s.Flush()
	if err != nil {
		return fmt.Errorf("failed to flush cache: %v", err)
	}
	err = s.proto.WriteLocked(false, s.file)
	if err != nil {
		return fmt.Errorf("failed to lock the file: %v", err)
	}
	err = s.file.Close()
	if err != nil {
		return fmt.Errorf("failed to close the underlying file: %v", err)
	}
	return nil
}

// Returns the number of items currently stored in the storage.
func (s *storage) GetUserCount() int {
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
func (s *storage) GetTotalItemCount() int {
	return s.totalItemCount
}

// Returns the storage file size required to keep all the data.
func (s *storage) GetFileSize() uint64 {
	return uint64(s.GetTotalItemCount())*entrySize + headerSize + uint64(len(prefix))
}

// Returns the last operation associated with the specified user-item pair.
// This method exists mostly for debugging and testing purposes.
func (s *storage) Get(user uint64, item uint64) (Operation, bool) {
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
func (s *storage) Add(op Operation, user uint64, item uint64) {
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
