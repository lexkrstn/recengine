package index

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"recengine/internal/domain"
)

// Implements database index storage.
type storage struct {
	file    io.ReadWriteSeeker
	closer  io.Closer
	indices map[uint64]uint64
	proto   Protocol
}

// Compile-time type check
var _ = (domain.IndexStorage)((*storage)(nil))

// Creates a new index file by the specified path.
func (s *storage) create() error {
	writer := bufio.NewWriter(s.file)
	// Prefix
	_, err := s.proto.WritePrefix(writer)
	if err != nil {
		return fmt.Errorf("failed to write index prefix: %v", err)
	}
	// Header
	header := Header{Version, 1, 0}
	_, err = s.proto.WriteHeader(&header, writer)
	if err != nil {
		return fmt.Errorf("failed to write index header: %v", err)
	}
	// Flush
	err = writer.Flush()
	if err != nil {
		return fmt.Errorf("failed to flush writer: %v", err)
	}
	return nil
}

// Loads the index file into memory.
func (s *storage) load() error {
	_, err := s.file.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}
	reader := bufio.NewReader(s.file)
	// Load header
	_, err = s.proto.ReadPrefix(reader)
	if err != nil {
		return fmt.Errorf("failed to read index prefix: %v", err)
	}
	header := Header{}
	_, err = s.proto.ReadHeader(&header, reader)
	if err != nil {
		return fmt.Errorf("failed to read index header: %v", err)
	}
	if header.Locked != 0 {
		return errors.New("broken index file (after unexpected termination)")
	}
	// Lock
	err = s.proto.WriteLocked(true, s.file)
	if err != nil {
		return fmt.Errorf("failed to write file lock: %v", err)
	}
	// Load entries
	entry := Entry{}
	offset := int64(entriesOffset)
	var size int
	for i := uint(0); i < uint(header.NumEntries); i++ {
		size, err = s.proto.ReadEntry(&entry, reader)
		offset += int64(size)
		if err != nil {
			return fmt.Errorf("failed to read index entry: %v", err)
		}
		s.indices[entry.ID] = entry.Index
	}
	return nil
}

// Closes the storage file.
func (s *storage) Close() error {
	err := s.saveHeader(false)
	if err != nil {
		if s.closer != nil {
			s.closer.Close()
		}
		return err
	}
	err = s.saveEntries()
	if err != nil {
		if s.closer != nil {
			s.closer.Close()
		}
		return err
	}
	if s.closer != nil {
		return s.closer.Close()
	}
	return nil
}

// Saves the header corresponding to the current state of the storage.
func (s *storage) saveHeader(locked bool) error {
	lockedByte := byte(0)
	if locked {
		lockedByte = 1
	}
	header := Header{Version, lockedByte, uint32(len(s.indices))}
	_, err := s.file.Seek(int64(len(prefix)), io.SeekStart)
	if err != nil {
		if s.closer != nil {
			s.closer.Close()
		}
		return fmt.Errorf("failed to seek: %v", err)
	}
	writer := bufio.NewWriter(s.file)
	_, err = s.proto.WriteHeader(&header, writer)
	if err != nil {
		if s.closer != nil {
			s.closer.Close()
		}
		return fmt.Errorf("failed to write index header: %v", err)
	}
	err = writer.Flush()
	if err != nil {
		return fmt.Errorf("failed flush buffer: %v", err)
	}
	return nil
}

// Saves cached entries into the disk.
func (s *storage) saveEntries() error {
	_, err := s.file.Seek(int64(entriesOffset), io.SeekStart)
	if err != nil {
		return fmt.Errorf("failed to seek: %v", err)
	}
	writer := bufio.NewWriter(s.file)
	entry := Entry{}
	for id, idx := range s.indices {
		entry.ID = id
		entry.Index = idx
		_, err = s.proto.WriteEntry(&entry, writer)
		if err != nil {
			return fmt.Errorf("failed to write entry: %v", err)
		}
	}
	err = writer.Flush()
	if err != nil {
		return fmt.Errorf("failed flush buffer: %v", err)
	}
	return nil
}

// Returns the index associated with the specified ID.
func (s *storage) Get(id uint64) (uint64, bool) {
	idx, ok := s.indices[id]
	return idx, ok
}

// Associates an index with an ID.
func (s *storage) Put(id uint64, index uint64) error {
	// Update the in-memory map
	s.indices[id] = index
	return nil
}

// Removes an index from the database.
func (s *storage) Remove(id uint64) error {
	if _, hasIndex := s.indices[id]; !hasIndex {
		return nil
	}
	delete(s.indices, id)
	return nil
}
