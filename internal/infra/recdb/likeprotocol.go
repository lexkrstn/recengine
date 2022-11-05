package recdb

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"recengine/internal/domain/entities"
	"reflect"
)

// Binary ("like") implementation of IConcreteProtocol.
type LikeProtocol struct{}

// Compile-type type check
var _ = (IConcreteProtocol)((*LikeProtocol)(nil))

// Instantiates a LikeProtocol.
func NewLikeProtocol() IConcreteProtocol {
	return &LikeProtocol{}
}

// Reads entry data filling the `Entry.Data` struct.
// Returns the number of the bytes having read.
// The returned size can vary from 0 to `Entry.Capacity`.
func (p *LikeProtocol) ReadEntryData(entry *Entry, reader io.Reader) (int, error) {
	// Read user ID
	var userId uint64
	err := binary.Read(reader, binary.BigEndian, &userId)
	if err != nil {
		return 0, err
	}
	// Read likes
	var numLikes uint32
	err = binary.Read(reader, binary.BigEndian, &numLikes)
	if err != nil {
		return 0, err
	}
	likes := make([]uint64, numLikes)
	err = binary.Read(reader, binary.BigEndian, likes)
	if err != nil {
		return 0, err
	}
	// Read dislikes
	var numDislikes uint32
	err = binary.Read(reader, binary.BigEndian, &numDislikes)
	if err != nil {
		return 0, err
	}
	dislikes := make([]uint64, numDislikes)
	err = binary.Read(reader, binary.BigEndian, dislikes)
	if err != nil {
		return 0, err
	}
	// Update entry.Data
	entry.Data = &entities.Profile{
		UserID:   userId,
		Likes:    likes,
		Dislikes: dislikes,
	}
	// Check data integrity
	size, err := p.PredictDataSize(entry.Data)
	if err != nil {
		return 0, fmt.Errorf("cannot predict data size: %w", err)
	}
	if size > int(entry.Capacity-entryHeaderSize) {
		return 0, fmt.Errorf("entry's capacity=%d is less than data len=%d",
			entry.Capacity, size)
	}
	return size, nil
}

// Writes entry data from the `Entry.Data` field into the stream.
// Returns the number of the bytes having read.
// The data length cannot be greater than `Entry.Capacity`.
func (p *LikeProtocol) WriteEntryData(entry *Entry, writer io.Writer) (int, error) {
	// Get profile
	profile, ok := entry.Data.(*entities.Profile)
	if !ok {
		return 0, fmt.Errorf(
			"entry's data type doesn't represent like profile, got %s",
			reflect.TypeOf(entry.Data).Name(),
		)
	}
	// Write user ID
	err := binary.Write(writer, binary.BigEndian, profile.UserID)
	if err != nil {
		return 0, err
	}
	// Write likes
	err = binary.Write(writer, binary.BigEndian, uint32(len(profile.Likes)))
	if err != nil {
		return 0, err
	}
	err = binary.Write(writer, binary.BigEndian, profile.Likes)
	if err != nil {
		return 0, err
	}
	// Write dislikes
	err = binary.Write(writer, binary.BigEndian, uint32(len(profile.Dislikes)))
	if err != nil {
		return 0, err
	}
	err = binary.Write(writer, binary.BigEndian, profile.Dislikes)
	if err != nil {
		return 0, err
	}
	return p.PredictDataSize(profile)
}

// Returns the type code of the data stored in the `Entry.Data` field of
// entries of the database file type that is handled by this implementation.
func (p *LikeProtocol) GetEntryType() [8]byte {
	return [...]byte{'L', 'I', 'K', 'E', ' ', ' ', ' ', ' '}
}

// Returns the minimum number of bytes it the entry will span after serialization.
func (p *LikeProtocol) PredictDataSize(data any) (int, error) {
	profile, ok := data.(*entities.Profile)
	if !ok {
		return 0, errors.New("unknown data type")
	}
	return 8 + 4 + len(profile.Likes)*8 + 4 + len(profile.Dislikes)*8, nil
}
