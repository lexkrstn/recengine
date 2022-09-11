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
const HeaderSize = 1 + 8 + 4

// The size of the data in the entry that is stored before the EntryData.
const EntryHeaderSize = 4 + 1

// The database file prefix (aka "Magic number").
var Prefix = [...]byte{'R', 'E', 'C', 'D', 'B'}

// Offset from the start of the file to the first entry.
const EntriesOffset = len(Prefix) + HeaderSize

// Database file header is located at the begining of the database file
// just after the file type prefix (aka "Magic number").
type Header struct {
	Version    byte
	EntryType  [8]byte
	NumEntries uint32
}

type Entry struct {
	Capacity uint32
	Deleted  byte
	Data     any
}

// Creates an empty database file or rewrites any existing one that is located
// by the specified path. The second argument declares what entry type can be
// stored within the database file (e.g. like or rating profiles). The actual
// value is implementation-specific, but must occupy exactly 8 bytes.
// The function returns a file pointer opened in read-write mode.
func Create(fileName string, entryType [8]byte) (*os.File, error) {
	file, err := os.Create(fileName)
	if err != nil {
		return nil, fmt.Errorf("Failed to create RECDB file: %v", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	_, err = WritePrefix(writer)
	if err != nil {
		file.Close()
		return nil, err
	}
	header := Header{Version, entryType, 0}
	_, err = WriteHeader(&header, writer)
	if err != nil {
		file.Close()
		return nil, err
	}
	return file, nil
}

// Tries to open the database file or create it if it doesn't exist yet.
// The second argument declares what entry type can be  stored within the
// database file (e.g. like or rating profiles). The function returns a
// pointer to the file opened in read-only mode.
func Open(fileName string, entryType [8]byte) (*os.File, error) {
	file, err := os.Open(fileName)
	if err != nil {
		if os.IsNotExist(err) {
			newFile, err := Create(fileName, entryType)
			if err != nil {
				return nil, err
			}
			newFile.Close()
			return Open(fileName, entryType)
		}
		return nil, fmt.Errorf("Failed to open database file: %v", err)
	}
	return file, err
}

// Writes database prefix, aka "Magic number", which verifies type of the file.
func WritePrefix(writer io.Writer) (int, error) {
	n, err := writer.Write(Prefix[:])
	if err != nil {
		return 0, fmt.Errorf("Failed to write RECDB prefix: %v", err)
	}
	return n, err
}

// Writes database header (without the prefix).
func WriteHeader(header *Header, writer io.Writer) (int, error) {
	buffer := make([]byte, 0, HeaderSize)
	buffer = append(buffer, header.Version)
	buffer = append(buffer, header.EntryType[:]...)
	buffer = binary.BigEndian.AppendUint32(buffer, header.NumEntries)
	n, err := writer.Write(buffer)
	if err != nil {
		return 0, fmt.Errorf("Failed to write RECDB header: %v", err)
	}
	return n, err
}

// Reads database prefix, aka "Magic number", which verifies type of the file.
func ReadPrefix(reader io.Reader) (int, error) {
	buffer := make([]byte, 0, len(Prefix))
	n, err := helpers.ReadFullLength(buffer, reader)
	if err != nil {
		return n, fmt.Errorf("Failed to read database prefix: %v", err)
	}
	if !reflect.DeepEqual(buffer, Prefix) {
		return n, fmt.Errorf("Failed to read database prefix: mismatch")
	}
	return n, nil
}

// Reads database header (without the prefix).
func ReadHeader(header *Header, reader io.Reader) (int, error) {
	buffer := make([]byte, 0, HeaderSize)
	n, err := helpers.ReadFullLength(buffer, reader)
	if err != nil {
		return n, fmt.Errorf("Failed to read database header: %v", err)
	}
	header.Version = buffer[0]
	header.EntryType = *(*[8]byte)(buffer[1:])
	header.NumEntries = binary.BigEndian.Uint32(buffer[9:])
	return n, nil
}

// A callback function for Write entry that writes the Data field into the stream.
type WriteDataFn = func(*Entry, io.Writer) (int, error)

// Writes a database entry. Returns number of bytes written.
// The last argument is a callback function that must write the type-specific
// Data field into the stream and return the number of bytes written.
func WriteEntry(entry *Entry, writer io.Writer, writeData WriteDataFn) (int, error) {
	const msg = "Failed to write database entry: %v"
	err := binary.Write(writer, binary.BigEndian, entry.Capacity)
	if err != nil {
		return 0, fmt.Errorf(msg, err)
	}
	err = binary.Write(writer, binary.BigEndian, entry.Deleted)
	if err != nil {
		return 0, fmt.Errorf(msg, err)
	}
	err = binary.Write(writer, binary.BigEndian, entry.Data)
	if err != nil {
		return 0, fmt.Errorf(msg, err)
	}
	var dataLen int
	dataLen, err = writeData(entry, writer)
	if err != nil {
		return 0, fmt.Errorf(msg, err)
	}
	numZeros := int(entry.Capacity) - EntryHeaderSize - dataLen
	if numZeros > 0 {
		_, err = helpers.WriteZeros(numZeros, writer)
		if err != nil {
			return 0, fmt.Errorf(msg, err)
		}
	}
	return int(entry.Capacity), nil
}

// A callback function for Write entry that writes the Data field into the stream.
type ReadDataFn = func(*Entry, io.Reader) (int, error)

// Reads a database entry. Returns number of bytes read.
// The last argument is a callback function that must read the type-specific
// Data field from the stream and return the number of bytes read.
func ReadEntry(entry *Entry, reader io.Reader, readData ReadDataFn) (int, error) {
	const msg = "Failed to read database entry: %v"
	// Read Capacity
	err := binary.Read(reader, binary.BigEndian, &entry.Capacity)
	if err != nil {
		return 0, fmt.Errorf(msg, err)
	}
	// Read Deleted
	var bytes [1]byte
	_, err = helpers.ReadFullLength(bytes[:], reader)
	if err != nil {
		return 0, fmt.Errorf(msg, err)
	}
	entry.Deleted = bytes[0]
	// Read EntryDataLen
	var dataLen int
	dataLen, err = readData(entry, reader)
	if err != nil {
		return 0, fmt.Errorf(msg, err)
	}
	// Seek free space
	numZeros := int(entry.Capacity) - EntryHeaderSize - dataLen
	_, err = helpers.Skip(numZeros, reader)
	if err != nil {
		return 0, fmt.Errorf(msg, err)
	}
	return int(entry.Capacity), nil
}
