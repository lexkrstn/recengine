package delta

import (
	"encoding/binary"
	"errors"
	"io"
	"recengine/internal/domain"
	"reflect"
)

// Delta file Header.
type Header struct {
	Version    uint8
	Locked     uint8
	NumEntries uint32
}

// Delta file Entry.
type Entry struct {
	Op       domain.DeltaOp
	UserID   uint64
	ItemID   uint64
	Checksum byte
}

// File format Version.
const Version = 1

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

// Provides delta file functions.
type Protocol interface {
	// Writes the file prefix, aka "Magic number", which verifies type of the file.
	WritePrefix(writer io.Writer) error

	// Reads the file prefix, aka "Magic number", which verifies type of the file.
	ReadPrefix(reader io.Reader) error

	// Writes file header (without the prefix).
	WriteHeader(header *Header, writer io.Writer) error

	// Reads database header (without the prefix).
	ReadHeader(header *Header, reader io.Reader) error

	// Calculates a byte checksum for an entry.
	CalcEntryChecksum(entry *Entry) byte

	// Writes a file entry. Returns number of bytes written.
	WriteEntry(entry *Entry, writer io.Writer) error

	// Reads a file entry. Returns the number of bytes read.
	ReadEntry(entry *Entry, reader io.Reader) error

	// Returns true if the checksum of the entry is valid or false otherwise.
	ValidateEntryChecksum(entry *Entry) bool

	// Writes the "locked" field of the file's header without changing file
	// pointer position. The file is considered corrupted if it's not unlocked,
	// which means it hasn't been closed properly.
	WriteLocked(locked bool, file io.WriteSeeker) error

	// Checks whether the file has the locked field set true without changing
	// the file pointer. The file is considered corrupted if it's not unlocked,
	// which means it hasn't been closed properly.
	IsLocked(file io.ReadSeeker) (bool, error)

	// Recovers a corrupted file making its data consistent. All inconsistent
	// data is skipped. The file is considered corrupted if it's locked, which
	// means it hasn't been closed properly.
	RecoverTo(reader io.Reader, writer io.WriteSeeker) error
}

// Implements delta file functions.
type protocol struct{}

// Compile-time type check
var _ = (Protocol)((*protocol)(nil))

// Returns new protocol instance.
func NewProtocol() Protocol {
	return &protocol{}
}

// Writes the file prefix, aka "Magic number", which verifies type of the file.
func (p *protocol) WritePrefix(writer io.Writer) error {
	_, err := writer.Write(prefix[:])
	return err
}

// Reads the file prefix, aka "Magic number", which verifies type of the file.
func (p *protocol) ReadPrefix(reader io.Reader) error {
	buffer := make([]byte, len(prefix))
	_, err := io.ReadFull(reader, buffer)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(buffer, prefix[:]) {
		return errors.New("not a delta file")
	}
	return nil
}

// Writes file header (without the prefix).
func (p *protocol) WriteHeader(header *Header, writer io.Writer) error {
	err := binary.Write(writer, binary.BigEndian, header.Version)
	if err != nil {
		return err
	}
	err = binary.Write(writer, binary.BigEndian, header.Locked)
	if err != nil {
		return err
	}
	return binary.Write(writer, binary.BigEndian, header.NumEntries)
}

// Reads database header (without the prefix).
func (p *protocol) ReadHeader(header *Header, reader io.Reader) error {
	buffer := make([]byte, headerSize)
	_, err := io.ReadFull(reader, buffer)
	if err != nil {
		return err
	}
	header.Version = buffer[0]
	header.Locked = buffer[1]
	header.NumEntries = binary.BigEndian.Uint32(buffer[2:])
	return nil
}

// Returns a byte checksum for a qword value.
func (p *protocol) calcUint64Checksum(n uint64) byte {
	b := make([]byte, 0, 8)
	b = binary.BigEndian.AppendUint64(b, n)
	var sum byte = 0
	for i := 0; i < 8; i++ {
		sum += b[i]
	}
	return sum
}

// Calculates a byte checksum for an entry.
func (p *protocol) CalcEntryChecksum(entry *Entry) byte {
	return byte(entry.Op) + p.calcUint64Checksum(entry.UserID) + p.calcUint64Checksum(entry.ItemID)
}

// Writes a file entry. Returns number of bytes written.
func (p *protocol) WriteEntry(entry *Entry, writer io.Writer) error {
	err := binary.Write(writer, binary.BigEndian, entry.Op)
	if err != nil {
		return err
	}
	err = binary.Write(writer, binary.BigEndian, entry.UserID)
	if err != nil {
		return err
	}
	err = binary.Write(writer, binary.BigEndian, entry.ItemID)
	if err != nil {
		return err
	}
	return binary.Write(writer, binary.BigEndian, p.CalcEntryChecksum(entry))
}

// Reads a file entry. Returns the number of bytes read.
func (p *protocol) ReadEntry(entry *Entry, reader io.Reader) error {
	err := binary.Read(reader, binary.BigEndian, &entry.Op)
	if err != nil {
		return err
	}
	err = binary.Read(reader, binary.BigEndian, &entry.UserID)
	if err != nil {
		return err
	}
	err = binary.Read(reader, binary.BigEndian, &entry.ItemID)
	if err != nil {
		return err
	}
	return binary.Read(reader, binary.BigEndian, &entry.Checksum)
}

// Returns true if the checksum of the entry is valid or false otherwise.
func (p *protocol) ValidateEntryChecksum(entry *Entry) bool {
	return p.CalcEntryChecksum(entry) == entry.Checksum
}

// Writes the "locked" field of the file's header without changing file
// pointer position. The file is considered corrupted if it's not unlocked,
// which means it hasn't been closed properly.
func (p *protocol) WriteLocked(locked bool, file io.WriteSeeker) error {
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
// the file pointer. The file is considered corrupted if it's not unlocked,
// which means it hasn't been closed properly.
func (p *protocol) IsLocked(file io.ReadSeeker) (bool, error) {
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

// Recovers a corrupted file making its data consistent. All inconsistent
// data is skipped. The file is considered corrupted if it's locked, which
// means it hasn't been closed properly.
func (p *protocol) RecoverTo(reader io.Reader, writer io.WriteSeeker) error {
	hdr := Header{
		Version:    Version,
		Locked:     0,
		NumEntries: 0,
	}

	err := p.WritePrefix(writer)
	if err != nil {
		return err
	}

	// Try to read prefix
	err = p.ReadPrefix(reader)
	if err != nil {
		return p.WriteHeader(&hdr, writer)
	}

	// Try to read header
	err = p.ReadHeader(&hdr, reader)
	hdr.Version = Version
	hdr.Locked = 0
	if err != nil {
		hdr.NumEntries = 0
		return p.WriteHeader(&hdr, writer)
	}
	err = p.WriteHeader(&hdr, writer)
	if err != nil {
		return err
	}

	// Copy valid entries
	entry := Entry{}
	var entriesRecovered uint32 = 0
	for {
		err = p.ReadEntry(&entry, reader)
		if err != nil {
			break
		}
		if p.CalcEntryChecksum(&entry) != entry.Checksum {
			continue
		}
		err = p.WriteEntry(&entry, writer)
		if err != nil {
			return err
		}
		entriesRecovered++
	}

	// Update entry count in the destination file's header
	if hdr.NumEntries != entriesRecovered {
		_, err = writer.Seek(int64(len(prefix)), io.SeekStart)
		if err != nil {
			return err
		}
		hdr.NumEntries = entriesRecovered
		return p.WriteHeader(&hdr, writer)
	}

	return nil
}
