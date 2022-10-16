package delta

import (
	"encoding/binary"
	"errors"
	"io"
	"recengine/internal/helpers"
	"reflect"
)

// Delta file header.
type header struct {
	version    uint8
	locked     uint8
	numEntries uint32
}

// OpAdd or OpRemove
type Operation byte

const (
	OpAdd    Operation = '+'
	OpRemove Operation = '-'
)

// Delta file entry.
type entry struct {
	op       Operation
	user     uint64
	item     uint64
	checksum byte
}

// File format version.
const version = 1

// Header size in bytes (the prefix is not part of the header).
// WARNING: because of the padding the header size may not equal sizeof(header)!
const headerSize = 1 + 1 + 4

// Entry size in bytes.
// WARNING: because of the padding the entry size may not equal sizeof(entry)!
const entrySize = 1 + 8 + 8 + 1

// The file prefix (aka "Magic number").
var prefix = [...]byte{'R', 'E', 'C', 'D', 'E', 'L', 'T', 'A'}

// The offset of the lock byte of the header from the beginning of the file.
const lockedOffset = len(prefix) + 1

// Writes the file prefix, aka "Magic number", which verifies type of the file.
func writePrefix(writer io.Writer) error {
	_, err := writer.Write(prefix[:])
	return err
}

// Reads the file prefix, aka "Magic number", which verifies type of the file.
func readPrefix(reader io.Reader) error {
	buffer := make([]byte, len(prefix))
	_, err := helpers.ReadFullLength(buffer, reader)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(buffer, prefix[:]) {
		return errors.New("Not a delta file")
	}
	return nil
}

// Writes file header (without the prefix).
func writeHeader(header *header, writer io.Writer) error {
	err := binary.Write(writer, binary.BigEndian, header.version)
	if err != nil {
		return err
	}
	err = binary.Write(writer, binary.BigEndian, header.locked)
	if err != nil {
		return err
	}
	return binary.Write(writer, binary.BigEndian, header.numEntries)
}

// Reads database header (without the prefix).
func readHeader(header *header, reader io.Reader) error {
	buffer := make([]byte, headerSize)
	_, err := helpers.ReadFullLength(buffer, reader)
	if err != nil {
		return err
	}
	header.version = buffer[0]
	header.locked = buffer[1]
	header.numEntries = binary.BigEndian.Uint32(buffer[2:])
	return nil
}

// Returns a byte checksum for a qword value.
func calcUint64Checksum(n uint64) byte {
	b := make([]byte, 0, 8)
	b = binary.BigEndian.AppendUint64(b, n)
	var sum byte = 0
	for i := 0; i < 8; i++ {
		sum += b[i]
	}
	return sum
}

// Calculates a byte checksum for an entry.
func calcEntryChecksum(entry *entry) byte {
	return byte(entry.op) + calcUint64Checksum(entry.user) + calcUint64Checksum(entry.item)
}

// Writes a file entry. Returns number of bytes written.
func writeEntry(entry *entry, writer io.Writer) error {
	err := binary.Write(writer, binary.BigEndian, entry.op)
	if err != nil {
		return err
	}
	err = binary.Write(writer, binary.BigEndian, entry.user)
	if err != nil {
		return err
	}
	err = binary.Write(writer, binary.BigEndian, entry.item)
	if err != nil {
		return err
	}
	return binary.Write(writer, binary.BigEndian, calcEntryChecksum(entry))
}

// Reads a file entry. Returns the number of bytes read.
func readEntry(entry *entry, reader io.Reader) error {
	err := binary.Read(reader, binary.BigEndian, &entry.op)
	if err != nil {
		return err
	}
	err = binary.Read(reader, binary.BigEndian, &entry.user)
	if err != nil {
		return err
	}
	err = binary.Read(reader, binary.BigEndian, &entry.item)
	if err != nil {
		return err
	}
	return binary.Read(reader, binary.BigEndian, &entry.checksum)
}

// Returns true if the checksum of the entry is valid or false otherwise.
func validateEntryChecksum(entry *entry) bool {
	return calcEntryChecksum(entry) == entry.checksum
}

// Writes the "locked" field of the file's header without changing file
// pointer position.
// The file is considered corrupted if it's not unlocked, which means it hasn't
// been closed properly.
func writeLocked(locked bool, file io.WriteSeeker) error {
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

// Checks whether the file has the locked field set true without changing
// the file pointer.
// The file is considered corrupted if it's not unlocked, which means it hasn't
// been closed properly.
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

// Recovers a corrupted file making its data consistent.
// All inconsistent data is skipped.
// The file is considered corrupted if it's locked, which means it hasn't been
// closed properly.
func recoverTo(reader io.Reader, writer io.WriteSeeker) error {
	hdr := header{
		version:    version,
		locked:     0,
		numEntries: 0,
	}

	err := writePrefix(writer)
	if err != nil {
		return err
	}

	// Try to read prefix
	err = readPrefix(reader)
	if err != nil {
		return writeHeader(&hdr, writer)
	}

	// Try to read header
	err = readHeader(&hdr, reader)
	hdr.version = version
	hdr.locked = 0
	if err != nil {
		hdr.numEntries = 0
		return writeHeader(&hdr, writer)
	}
	err = writeHeader(&hdr, writer)
	if err != nil {
		return err
	}

	// Copy valid entries
	entry := entry{}
	var entriesRecovered uint32 = 0
	for {
		err = readEntry(&entry, reader)
		if err != nil {
			break
		}
		if calcEntryChecksum(&entry) != entry.checksum {
			continue
		}
		err = writeEntry(&entry, writer)
		if err != nil {
			return err
		}
		entriesRecovered++
	}

	// Update entry count in the destination file's header
	if hdr.numEntries != entriesRecovered {
		_, err = writer.Seek(int64(len(prefix)), io.SeekStart)
		if err != nil {
			return err
		}
		hdr.numEntries = entriesRecovered
		return writeHeader(&hdr, writer)
	}

	return nil
}
