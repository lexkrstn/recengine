package index

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

// Database index storage.
type IndexStorage struct {
	file       *os.File
	indices    map[uint64]uint64
	offsets    map[uint64]int64
	writer     *bufio.Writer
	numDeleted int
}

// Opens an index file by the specified path. If the file doesn't exist yet
// it will be created.
func Open(filePath string) (*IndexStorage, error) {
	file, err := os.OpenFile(filePath, os.O_RDWR, 0644)
	if err != nil {
		if os.IsNotExist(err) {
			file, err = create(filePath)
			if err != nil {
				return nil, fmt.Errorf("Failed to create index file: %v", err)
			}
		} else {
			return nil, err
		}
	}
	storage := &IndexStorage{
		file,
		make(map[uint64]uint64),
		make(map[uint64]int64),
		bufio.NewWriter(file),
		0,
	}
	err = storage.load()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("Failed to load index: %v", err)
	}
	return storage, nil
}

// Creates a new index file by the specified path.
func create(filePath string) (*os.File, error) {
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, fmt.Errorf("Failed to open index file: %v", err)
	}
	writer := bufio.NewWriter(file)
	// Prefix
	_, err = writePrefix(writer)
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("Failed to write index prefix: %v", err)
	}
	// Header
	header := header{version, 0, 0}
	_, err = writeHeader(&header, writer)
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("Failed to write index header: %v", err)
	}
	// Flush
	err = writer.Flush()
	if err != nil {
		file.Close()
		return nil, err
	}
	return file, nil
}

// Loads the index file into memory.
func (s *IndexStorage) load() error {
	_, err := s.file.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}
	reader := bufio.NewReader(s.file)
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
		return fmt.Errorf("Broken index file: %s", s.file.Name())
	}
	err = setLocked(true, s.file)
	if err != nil {
		return fmt.Errorf("Failed to set file lock: %v", err)
	}
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
	header := header{version, 0, uint32(len(s.offsets))}
	_, err := s.file.Seek(int64(len(prefix)), io.SeekStart)
	if err != nil {
		s.file.Close()
		return fmt.Errorf("Failed to seek in %s: %v", s.file.Name(), err)
	}
	_, err = writeHeader(&header, s.file)
	if err != nil {
		s.file.Close()
		return fmt.Errorf("Failed to write index header: %v", err)
	}
	return s.file.Close()
}

// Returns the index associated with the specified ID.
func (s *IndexStorage) GetIndex(id uint64) (uint64, bool) {
	idx, ok := s.indices[id]
	return idx, ok
}

// Associates an index with an ID.
func (s *IndexStorage) PutIndex(id uint64, index uint64) error {
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
func (s *IndexStorage) RemoveIndex(id uint64) error {
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
