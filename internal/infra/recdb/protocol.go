package recdb

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"recengine/internal/helpers"
	"reflect"
)

// Current database file format version.
const Version byte = 1

// The size in bytes, that occupies the header of the database file.
const headerSize = 1 + 8 + 1 + 4

// The size of the data in the entry that is stored before the EntryData.
const entryHeaderSize = 4 + 1

// The database file prefix (aka "Magic number").
var prefix = [...]byte{'R', 'E', 'C', 'D', 'B'}

// Offset from the start of the file to the first entry.
const entriesOffset = len(prefix) + headerSize

// The offset of the lock byte of the header from the beginning of the file.
const lockedOffset = len(prefix) + 1 + 8

// Minimum entry's capacity including entry header size.
const minEntryCapacity = entryHeaderSize + 8*30

// Database file header is located at the begining of the database file
// just after the file type prefix (aka "Magic number").
type Header struct {
	// File protocol version.
	Version byte

	// Specified the type of the entries stored in the file.
	EntryType [8]byte

	// The file is locked when opened and unlocked during closing.
	// A file being locked upon opening means the program was terminated
	// abnormally, and the file must be checked for corruption and recovered.
	// 1 - locked, 0 - unlocked.
	Locked byte

	// Number of entries stored in the file.
	NumEntries uint32
}

// Database entry record.
type Entry struct {
	// Serialized entry's size including the size of this field itself.
	Capacity uint32

	// 0 - the entry is not deleted, 1 - deleted.
	Deleted byte

	// A pointer to a concrete-implementation-dependent structure.
	Data any
}

// Concrete protocol implementations (e.g. LikeProtocol) must implement this interface.
type ConcreteProtocol interface {
	// Reads entry data filling the `Entry.Data` struct.
	// Returns the number of the bytes having read.
	// The returned size can vary from 0 to `Entry.Capacity`.
	ReadEntryData(*Entry, io.Reader) (int, error)

	// Writes entry data from the `Entry.Data` field into the stream.
	// Returns the number of the bytes having read.
	// The data length cannot be greater than `Entry.Capacity`.
	WriteEntryData(*Entry, io.Writer) (int, error)

	// Returns the type code of the data stored in the `Entry.Data` field of
	// entries of the database file type that is handled by this implementation.
	GetEntryType() [8]byte

	// Returns the minimum number of bytes it the entry will span after serialization.
	PredictDataSize(data any) (int, error)
}

// Provides recommendation DB file functions.
// The protocol doesn't know about concrete implementation of Entry data.
type Protocol interface {
	// Initializes an empty database.
	Create(writer io.Writer) error

	// Writes database prefix, aka "Magic number", which verifies type of the file.
	WritePrefix(writer io.Writer) (int, error)

	// Reads database prefix, aka "Magic number", which verifies type of the file.
	ReadPrefix(reader io.Reader) (int, error)

	// Writes database header (without the prefix).
	WriteHeader(header *Header, writer io.Writer) (int, error)

	// Reads database header (without the prefix).
	ReadHeader(header *Header, reader io.Reader) (int, error)

	// Writes a database entry. Returns number of bytes written.
	// The last argument is a callback function that must write the type-specific
	// Data field into the stream and return the number of bytes written.
	WriteEntry(entry *Entry, writer io.Writer) (int, error)

	// Reads a database entry. Returns number of bytes read.
	// The last argument is a callback function that must read the type-specific
	// Data field from the stream and return the number of bytes read.
	ReadEntry(entry *Entry, reader io.Reader) (int, error)

	// Writes the "locked" field of the file's header without changing file
	// pointer position.
	WriteLocked(locked bool, file io.WriteSeeker) error

	// Checks whether the index file has the locked field set true without
	// changing the file pointer.
	IsLocked(file io.ReadSeeker) (bool, error)

	// Returns the minimum number of bytes it the entry will span after serialization.
	PredictEntrySize(entry *Entry) (int, error)

	// Returns the optimal capacity for the entry in bytes.
	PredictEntryCapacity(entry *Entry) (int, error)
}

// Implements abstract recommendation DB file functions.
// The protocol doesn't know about concrete implementation of Entry data.
type protocol struct {
	concreteProto ConcreteProtocol
}

// Compile-time type check
var _ = (Protocol)((*protocol)(nil))

// Instantiates a new protocol functions implementation using a concrete
// implementation of reading and writing entry data.
func NewProtocol(concreteProto ConcreteProtocol) Protocol {
	return &protocol{
		concreteProto: concreteProto,
	}
}

// Initializes an empty database.
func (p *protocol) Create(writer io.Writer) error {
	bufWriter := bufio.NewWriter(writer)
	_, err := p.WritePrefix(bufWriter)
	if err != nil {
		return err
	}
	header := Header{Version, p.concreteProto.GetEntryType(), 0, 0}
	_, err = p.WriteHeader(&header, bufWriter)
	if err != nil {
		return err
	}
	return nil
}

// Tries to open the database file or create it if it doesn't exist yet.
// The second argument declares what entry type can be  stored within the
// database file (e.g. like or rating profiles). The function returns a
// pointer to the file opened in read-only mode.
func (p *protocol) OpenOrCreateFile(filePath string) (*os.File, error) {
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			file, err := os.Create(filePath)
			if err != nil {
				return nil, fmt.Errorf("failed to create RECDB file: %v", err)
			}
			err = p.Create(file)
			if err != nil {
				file.Close()
				return nil, err
			}
			file.Seek(0, io.SeekStart)
			return file, nil
		}
		return nil, fmt.Errorf("failed to open database file: %v", err)
	}
	return file, err
}

// Writes database prefix, aka "Magic number", which verifies type of the file.
func (p *protocol) WritePrefix(writer io.Writer) (int, error) {
	n, err := writer.Write(prefix[:])
	if err != nil {
		return 0, fmt.Errorf("failed to write RECDB prefix: %v", err)
	}
	return n, err
}

// Reads database prefix, aka "Magic number", which verifies type of the file.
// Returns an error if the prefix is incorrect.
func (p *protocol) ReadPrefix(reader io.Reader) (int, error) {
	buffer := make([]byte, len(prefix))
	n, err := io.ReadFull(reader, buffer)
	if err != nil {
		return n, err
	}
	if !reflect.DeepEqual(buffer, prefix[:]) {
		return n, fmt.Errorf("invalid prefix")
	}
	return n, nil
}

// Writes database header (without the prefix).
func (p *protocol) WriteHeader(header *Header, writer io.Writer) (int, error) {
	buffer := make([]byte, 0, headerSize)
	buffer = append(buffer, header.Version)
	buffer = append(buffer, header.EntryType[:]...)
	buffer = append(buffer, header.Locked)
	buffer = binary.BigEndian.AppendUint32(buffer, header.NumEntries)
	n, err := writer.Write(buffer)
	if err != nil {
		return 0, fmt.Errorf("failed to write RECDB header: %v", err)
	}
	return n, err
}

// Reads database header (without the prefix).
func (p *protocol) ReadHeader(header *Header, reader io.Reader) (int, error) {
	buffer := make([]byte, headerSize)
	n, err := io.ReadFull(reader, buffer)
	if err != nil {
		return n, fmt.Errorf("failed to read database header: %v", err)
	}
	header.Version = buffer[0]
	header.EntryType = *(*[8]byte)(buffer[1:])
	header.Locked = buffer[9]
	header.NumEntries = binary.BigEndian.Uint32(buffer[10:])
	return n, nil
}

// Writes a database entry. Returns number of bytes written.
// The last argument is a callback function that must write the type-specific
// Data field into the stream and return the number of bytes written.
func (p *protocol) WriteEntry(entry *Entry, writer io.Writer) (int, error) {
	const msg = "failed to write database entry: %w"
	err := binary.Write(writer, binary.BigEndian, entry.Capacity)
	if err != nil {
		return 0, fmt.Errorf(msg, err)
	}
	err = binary.Write(writer, binary.BigEndian, entry.Deleted)
	if err != nil {
		return 0, fmt.Errorf(msg, err)
	}
	var dataLen int
	dataLen, err = p.concreteProto.WriteEntryData(entry, writer)
	if err != nil {
		return 0, fmt.Errorf(msg, err)
	}
	numZeros := int(entry.Capacity) - entryHeaderSize - dataLen
	if numZeros > 0 {
		_, err = helpers.WriteZeros(numZeros, writer)
		if err != nil {
			return 0, fmt.Errorf(msg, err)
		}
	}
	return int(entry.Capacity), nil
}

// Reads a database entry. Returns number of bytes read.
// The last argument is a callback function that must read the type-specific
// Data field from the stream and return the number of bytes read.
func (p *protocol) ReadEntry(entry *Entry, reader io.Reader) (int, error) {
	const msg = "failed to read database entry: %v"
	// Read Capacity
	err := binary.Read(reader, binary.BigEndian, &entry.Capacity)
	if err != nil {
		return 0, fmt.Errorf(msg, err)
	}
	// Read Deleted
	var bytes [1]byte
	_, err = io.ReadFull(reader, bytes[:])
	if err != nil {
		return 0, fmt.Errorf(msg, err)
	}
	entry.Deleted = bytes[0]
	// Read EntryDataLen
	var dataLen int
	dataLen, err = p.concreteProto.ReadEntryData(entry, reader)
	if err != nil {
		return 0, fmt.Errorf(msg, err)
	}
	// Seek free space
	numZeros := int(entry.Capacity) - entryHeaderSize - dataLen
	_, err = helpers.SkipReading(numZeros, reader)
	if err != nil {
		return 0, fmt.Errorf(msg, err)
	}
	return int(entry.Capacity), nil
}

// Writes the "locked" field of the file's header without changing file
// pointer position.
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

// Checks whether the index file has the locked field set true without changing
// the file pointer.
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

// Returns the minimum number of bytes it the entry will span after serialization.
func (p *protocol) PredictEntrySize(entry *Entry) (int, error) {
	size, err := p.concreteProto.PredictDataSize(entry.Data)
	if err != nil {
		return 0, fmt.Errorf("cannot predict data size: %w", err)
	}
	return size + entryHeaderSize, nil
}

// Returns the optimal capacity for the entry in bytes.
func (p *protocol) PredictEntryCapacity(entry *Entry) (int, error) {
	size, err := p.PredictEntrySize(entry)
	if err != nil {
		return 0, fmt.Errorf("cannot predict entry size: %w", err)
	}
	if size < minEntryCapacity {
		return minEntryCapacity, nil
	}
	return size + size/2, nil
}
