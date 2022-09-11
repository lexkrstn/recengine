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
	file       io.ReadWriteSeeker
	closer     io.Closer
	indices    map[uint64]uint64
	offsets    map[uint64]int64
	writer     *bufio.Writer
	numDeleted int
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
		make(map[uint64]int64),
		bufio.NewWriter(file),
		0,
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
		s.offsets[entry.id] = offset
		if entry.deleted != 0 {
			s.numDeleted++
		} else {
			s.indices[entry.id] = entry.index
		}
	}
	return nil
}

// Closes the storage file.
func (s *IndexStorage) Close() error {
	s.saveHeader(false)
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
	header := header{version, lockedByte, uint32(len(s.offsets))}
	_, err := s.file.Seek(int64(len(prefix)), io.SeekStart)
	if err != nil {
		if s.closer != nil {
			s.closer.Close()
		}
		return fmt.Errorf("Failed to seek: %v", err)
	}
	_, err = writeHeader(&header, s.file)
	if err != nil {
		if s.closer != nil {
			s.closer.Close()
		}
		return fmt.Errorf("Failed to write index header: %v", err)
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
	_, hadIndex := s.indices[id]
	s.indices[id] = index
	// Update the file
	offset, exists := s.offsets[id]
	var err error
	if !exists {
		offset, err = s.file.Seek(0, io.SeekEnd)
		s.offsets[id] = offset
	} else {
		_, err = s.file.Seek(offset, io.SeekStart)
		if !hadIndex { // Undeleting
			s.numDeleted--
		}
	}
	if err != nil {
		return err
	}
	s.writer.Reset(s.file)
	entry := entry{0, id, index}
	_, err = writeEntry(&entry, s.writer)
	if err != nil {
		return fmt.Errorf("Failed to write index entry: %v", err)
	}
	return s.writer.Flush()
}

// Removes an index from the database.
func (s *IndexStorage) Remove(id uint64) error {
	if _, hasIndex := s.indices[id]; !hasIndex {
		return nil
	}
	// Update the in-memory map
	delete(s.indices, id)
	// Update the file
	offset, exists := s.offsets[id]
	if !exists {
		return nil
	}
	_, err := s.file.Seek(offset, io.SeekStart)
	if err != nil {
		return err
	}
	err = writeEntryDeleted(true, s.file)
	if err != nil {
		return fmt.Errorf("Failed to write entry deleted flag: %v", err)
	}
	s.numDeleted++
	return nil
}
