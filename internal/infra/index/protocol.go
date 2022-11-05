package index

import (
	"encoding/binary"
	"errors"
	"io"
	"reflect"
)

// Database index file format Version.
const Version = 1

// Header size in bytes (the prefix is not part of the header).
const headerSize = 1 + 1 + 4

// Entry size in bytes
const entrySize = 8 + 8

// The file prefix (aka "Magic number").
var prefix = [...]byte{'R', 'E', 'C', 'I', 'D', 'X'}

// The offset of the first entry byte from the beginning of the file.
const entriesOffset = len(prefix) + headerSize

// The offset of the lock byte of the header from the beginning of the file.
const lockedOffset = len(prefix) + 1

// Database index file Header.
type Header struct {
	Version    uint8
	Locked     uint8
	NumEntries uint32
}

// Database index record.
type Entry struct {
	ID    uint64
	Index uint64
}

// Provides index file functions.
type IProtocol interface {
	WritePrefix(writer io.Writer) (int, error)
	ReadPrefix(reader io.Reader) (int, error)
	WriteHeader(header *Header, writer io.Writer) (int, error)
	ReadHeader(header *Header, reader io.Reader) (int, error)
	WriteEntry(entry *Entry, writer io.Writer) (int, error)
	ReadEntry(entry *Entry, reader io.Reader) (int, error)
	WriteLocked(locked bool, file io.WriteSeeker) error
	IsLocked(file io.ReadSeeker) (bool, error)
}

// Implements delta file functions.
type Protocol struct{}

// Compile-type type check
var _ = (IProtocol)((*Protocol)(nil))

// Writes the file prefix, aka "Magic number", which verifies type of the file.
func (p *Protocol) WritePrefix(writer io.Writer) (int, error) {
	n, err := writer.Write(prefix[:])
	if err != nil {
		return 0, err
	}
	return n, nil
}

// Reads the file prefix, aka "Magic number", which verifies type of the file.
func (p *Protocol) ReadPrefix(reader io.Reader) (int, error) {
	buffer := make([]byte, len(prefix))
	n, err := io.ReadFull(reader, buffer)
	if err != nil {
		return n, err
	}
	if !reflect.DeepEqual(buffer, prefix[:]) {
		return n, errors.New("not an index file")
	}
	return n, nil
}

// Writes file header (without the prefix).
func (p *Protocol) WriteHeader(header *Header, writer io.Writer) (int, error) {
	err := binary.Write(writer, binary.BigEndian, header.Version)
	if err != nil {
		return 0, err
	}
	err = binary.Write(writer, binary.BigEndian, header.Locked)
	if err != nil {
		return 0, err
	}
	err = binary.Write(writer, binary.BigEndian, header.NumEntries)
	if err != nil {
		return 0, err
	}
	return headerSize, nil
}

// Reads database header (without the prefix).
func (p *Protocol) ReadHeader(header *Header, reader io.Reader) (int, error) {
	buffer := make([]byte, headerSize)
	n, err := io.ReadFull(reader, buffer)
	if err != nil {
		return n, err
	}
	header.Version = buffer[0]
	header.Locked = buffer[1]
	header.NumEntries = binary.BigEndian.Uint32(buffer[2:])
	return n, nil
}

// Writes a file entry. Returns number of bytes written.
func (p *Protocol) WriteEntry(entry *Entry, writer io.Writer) (int, error) {
	err := binary.Write(writer, binary.BigEndian, entry.ID)
	if err != nil {
		return 0, err
	}
	err = binary.Write(writer, binary.BigEndian, entry.Index)
	if err != nil {
		return 0, err
	}
	return entrySize, nil
}

// Reads a database entry. Returns the number of bytes read.
func (p *Protocol) ReadEntry(entry *Entry, reader io.Reader) (int, error) {
	// Read "id"
	err := binary.Read(reader, binary.BigEndian, &entry.ID)
	if err != nil {
		return 0, err
	}
	// Read "index"
	err = binary.Read(reader, binary.BigEndian, &entry.Index)
	if err != nil {
		return 0, err
	}
	return entrySize, nil
}

// Writes the "locked" field of the file's header without changing file
// pointer position.
func (p *Protocol) WriteLocked(locked bool, file io.WriteSeeker) error {
	pos, err := file.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}
	_, err = file.Seek(int64(lockedOffset), io.SeekStart)
	if err != nil {
		return err
	}
	bytes := []byte{0}
	if locked {
		bytes[0] = 1
	}
	_, err = file.Write(bytes)
	if err != nil {
		return err
	}
	_, err = file.Seek(pos, io.SeekStart)
	if err != nil {
		return err
	}
	return nil
}

// Checks whether the index file has the locked field set true without changing
// the file pointer.
func (p *Protocol) IsLocked(file io.ReadSeeker) (bool, error) {
	// Save current position
	pos, err := file.Seek(0, io.SeekCurrent)
	if err != nil {
		return false, err
	}
	// Read the byte that is responsible for locking
	_, err = file.Seek(int64(lockedOffset), io.SeekStart)
	if err != nil {
		return false, err
	}
	bytes := []byte{0}
	_, err = file.Read(bytes)
	if err != nil {
		return false, err
	}
	// Restore the initial position
	_, err = file.Seek(pos, io.SeekStart)
	if err != nil {
		return false, err
	}
	return bytes[0] != 0, err
}
