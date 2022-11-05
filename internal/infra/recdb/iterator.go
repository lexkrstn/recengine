package recdb

import (
	"bufio"
	"errors"
	"fmt"
	"io"
)

// The database entry iterator.
type Iterator struct {
	header         *Header
	file           io.ReadWriteSeeker
	reader         *bufio.Reader
	entryOffset    int
	fileOffset     int64
	previousOffset int64
	entry          Entry
	proto          IProtocol
}

// Creates a new database entry iterator.
func NewIterator(file io.ReadWriteSeeker, proto IProtocol) (*Iterator, error) {
	_, err := file.Seek(0, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to rewind DB iterator: %v", err)
	}
	_, err = proto.ReadPrefix(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read DB prefix: %v", err)
	}
	header := Header{}
	_, err = proto.ReadHeader(&header, file)
	if err != nil {
		return nil, fmt.Errorf("failed to read DB header: %v", err)
	}
	if header.Locked != 0 {
		return nil, NewLockedFileError()
	}
	reader := bufio.NewReader(file)
	iter := &Iterator{&header, file, reader, 0, 0, 0, Entry{}, proto}
	return iter, nil
}

// Sets the iterator pointer to the beginging of the entries.
func (iter *Iterator) Rewind() error {
	var err error
	iter.fileOffset, err = iter.file.Seek(int64(entriesOffset), 0)
	iter.entryOffset = 0
	iter.previousOffset = 0
	iter.reader.Reset(iter.file)
	if err != nil {
		return fmt.Errorf("failed to rewind DB iterator: %v", err)
	}
	return nil
}

// Returns true if there is at least one entry that can be got by GetNext().
func (iter *Iterator) HasNext() bool {
	return iter.entryOffset < int(iter.header.NumEntries)
}

// Returns the current entry and iterates next.
// The returned entry pointer shouldn't be changed or taken over: it points to
// the same struct in memory in every iteration.
func (iter *Iterator) GetNext() (*Entry, error) {
	if !iter.HasNext() {
		return nil, errors.New("failed to iterate next: the last entry reached")
	}
	if iter.entryOffset > 0 {
		iter.previousOffset = iter.fileOffset
	}
	size, err := iter.proto.ReadEntry(&iter.entry, iter.reader)
	iter.fileOffset += int64(size)
	if err != nil {
		return nil, fmt.Errorf("failed to iterate next: %v", err)
	}
	iter.entryOffset++
	return &iter.entry, nil
}

// Rewrites a previously read entry. The capacity must be left unchanged.
func (iter *Iterator) SetPrevious(entry *Entry) error {
	const msg = "failed to SetPrevious: %v"
	if iter.entryOffset == 0 {
		return errors.New("failed to set previous entry: at the start")
	}
	_, err := iter.file.Seek(iter.previousOffset, 0)
	if err != nil {
		return fmt.Errorf(msg, err)
	}
	_, err = iter.proto.WriteEntry(entry, iter.file)
	if err != nil {
		return fmt.Errorf(msg, err)
	}
	iter.reader.Reset(iter.file)
	return nil
}
