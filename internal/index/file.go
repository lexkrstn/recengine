package index

import (
	"encoding/binary"
	"errors"
	"io"
	"recengine/internal/helpers"
	"reflect"
)

// Database index file format version.
const version = 1

// Header size in bytes (the prefix is not part of the header).
const headerSize = 1 + 1 + 4

// Entry size in bytes
const entrySize = 1 + 8 + 8

// The file prefix (aka "Magic number").
var prefix = [...]byte{'R', 'E', 'C', 'I', 'D', 'X'}

const entriesOffset = len(prefix) + headerSize
const lockedOffset = len(prefix) + 1

// Database index file header.
type header struct {
	version    uint8
	locked     uint8
	numEntries uint32
}

// Database index table row.
type entry struct {
	deleted byte
	id      uint64
	index   uint64
}

// Writes the file prefix, aka "Magic number", which verifies type of the file.
func writePrefix(writer io.Writer) (int, error) {
	n, err := writer.Write(prefix[:])
	if err != nil {
		return 0, err
	}
	return n, nil
}

// Reads the file prefix, aka "Magic number", which verifies type of the file.
func readPrefix(reader io.Reader) (int, error) {
	buffer := make([]byte, len(prefix))
	n, err := helpers.ReadFullLength(buffer, reader)
	if err != nil {
		return n, err
	}
	if !reflect.DeepEqual(buffer, prefix[:]) {
		return n, errors.New("Not an index file")
	}
	return n, nil
}

// Writes file header (without the prefix).
func writeHeader(header *header, writer io.Writer) (int, error) {
	err := binary.Write(writer, binary.BigEndian, header.version)
	if err != nil {
		return 0, err
	}
	err = binary.Write(writer, binary.BigEndian, header.locked)
	if err != nil {
		return 0, err
	}
	err = binary.Write(writer, binary.BigEndian, header.numEntries)
	if err != nil {
		return 0, err
	}
	return headerSize, nil
}

// Reads database header (without the prefix).
func readHeader(header *header, reader io.Reader) (int, error) {
	buffer := make([]byte, headerSize)
	n, err := helpers.ReadFullLength(buffer, reader)
	if err != nil {
		return n, err
	}
	header.version = buffer[0]
	header.locked = buffer[1]
	header.numEntries = binary.BigEndian.Uint32(buffer[2:])
	return n, nil
}

// Writes a file entry. Returns number of bytes written.
func writeEntry(entry *entry, writer io.Writer) (int, error) {
	err := binary.Write(writer, binary.BigEndian, entry.deleted)
	if err != nil {
		return 0, err
	}
	err = binary.Write(writer, binary.BigEndian, entry.id)
	if err != nil {
		return 0, err
	}
	err = binary.Write(writer, binary.BigEndian, entry.index)
	if err != nil {
		return 0, err
	}
	return entrySize, nil
}

// Reads a database entry. Returns number of bytes read.
func readEntry(entry *entry, reader io.Reader) (int, error) {
	// Read "deleted"
	var bytes [1]byte
	_, err := helpers.ReadFullLength(bytes[:], reader)
	if err != nil {
		return 0, err
	}
	entry.deleted = bytes[0]
	// Read "id"
	err = binary.Read(reader, binary.BigEndian, &entry.id)
	if err != nil {
		return 0, err
	}
	// Read "index"
	err = binary.Read(reader, binary.BigEndian, &entry.index)
	if err != nil {
		return 0, err
	}
	return entrySize, nil
}

// Writes a file entry's deleted flag.
func writeEntryDeleted(deleted bool, writer io.Writer) error {
	return binary.Write(writer, binary.BigEndian, deleted)
}

// Writes the "locked" field of the file's header without changing file
// pointer position.
func WriteLocked(locked bool, file io.WriteSeeker) error {
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
func IsLocked(file io.ReadSeeker) (bool, error) {
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
