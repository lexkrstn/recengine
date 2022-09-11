package index

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
)

// Database index storage.
type IndexStorage struct {
	file    io.ReadWriteSeeker
	closer  io.Closer
	indices map[uint64]uint64
}

// Opens an index file by the specified path. If the file doesn't exist yet
// it will be created.
func OpenFile(filePath string) (*IndexStorage, error) {
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	store, err := Open(file, file)
	return store, err
}

// Opens an index file by the specified path. If the file doesn't exist yet
// it will be created.
func Open(file io.ReadWriteSeeker, closer io.Closer) (*IndexStorage, error) {
	storage := &IndexStorage{
		file,
		closer,
		make(map[uint64]uint64),
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
			return nil, fmt.Errorf("Failed to load index: %v", err)
		}
	} else {
		err = storage.create()
		if err != nil {
			if closer != nil {
				closer.Close()
			}
			return nil, fmt.Errorf("Failed to load index: %v", err)
		}
	}
	return storage, nil
}

// Creates a new index file by the specified path.
func (s *IndexStorage) create() error {
	writer := bufio.NewWriter(s.file)
	// Prefix
	_, err := writePrefix(writer)
	if err != nil {
		return fmt.Errorf("Failed to write index prefix: %v", err)
	}
	// Header
	header := header{version, 1, 0}
	_, err = writeHeader(&header, writer)
	if err != nil {
		return fmt.Errorf("Failed to write index header: %v", err)
	}
	// Flush
	err = writer.Flush()
	if err != nil {
		return fmt.Errorf("Failed to flush writer: %v", err)
	}
	return nil
}

// Loads the index file into memory.
func (s *IndexStorage) load() error {
	_, err := s.file.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}
	reader := bufio.NewReader(s.file)
	// Load header
	_, err = readPrefix(reader)
	if err != nil {
		return fmt.Errorf("Failed to read index prefix: %v", err)
	}
	header := header{}
	_, err = readHeader(&header, reader)
	if err != nil {
		return fmt.Errorf("Failed to read index header: %v", err)
	}
	if header.locked != 0 {
		return errors.New("Broken index file (after unexpected termination)")
	}
	// Lock
	err = WriteLocked(true, s.file)
	if err != nil {
		return fmt.Errorf("Failed to write file lock: %v", err)
	}
	// Load entries
	entry := entry{}
	offset := int64(entriesOffset)
	var size int
	for i := uint(0); i < uint(header.numEntries); i++ {
		size, err = readEntry(&entry, reader)
		offset += int64(size)
		if err != nil {
			return fmt.Errorf("Failed to read index entry: %v", err)
		}
		s.indices[entry.id] = entry.index
	}
	return nil
}

// Closes the storage file.
func (s *IndexStorage) Close() error {
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
func (s *IndexStorage) saveHeader(locked bool) error {
	lockedByte := byte(0)
	if locked {
		lockedByte = 1
	}
	header := header{version, lockedByte, uint32(len(s.indices))}
	_, err := s.file.Seek(int64(len(prefix)), io.SeekStart)
	if err != nil {
		if s.closer != nil {
			s.closer.Close()
		}
		return fmt.Errorf("Failed to seek: %v", err)
	}
	writer := bufio.NewWriter(s.file)
	_, err = writeHeader(&header, writer)
	if err != nil {
		if s.closer != nil {
			s.closer.Close()
		}
		return fmt.Errorf("Failed to write index header: %v", err)
	}
	err = writer.Flush()
	if err != nil {
		return fmt.Errorf("Failed flush buffer: %v", err)
	}
	return nil
}

// Saves cached entries into the disk.
func (s *IndexStorage) saveEntries() error {
	_, err := s.file.Seek(int64(entriesOffset), io.SeekStart)
	if err != nil {
		return fmt.Errorf("Failed to seek: %v", err)
	}
	writer := bufio.NewWriter(s.file)
	entry := entry{}
	for id, idx := range s.indices {
		entry.id = id
		entry.index = idx
		_, err = writeEntry(&entry, writer)
		if err != nil {
			return fmt.Errorf("Failed to write entry: %v", err)
		}
	}
	err = writer.Flush()
	if err != nil {
		return fmt.Errorf("Failed flush buffer: %v", err)
	}
	return nil
}

// Returns the index associated with the specified ID.
func (s *IndexStorage) Get(id uint64) (uint64, bool) {
	idx, ok := s.indices[id]
	return idx, ok
}

// Associates an index with an ID.
func (s *IndexStorage) Put(id uint64, index uint64) error {
	// Update the in-memory map
	s.indices[id] = index
	return nil
}

// Removes an index from the database.
func (s *IndexStorage) Remove(id uint64) error {
	if _, hasIndex := s.indices[id]; !hasIndex {
		return nil
	}
	delete(s.indices, id)
	return nil
}
