package recdb

import (
	"bytes"
	"recengine/internal/domain/entities"
	"recengine/internal/helpers"
	"reflect"
	"testing"
)

func TestLikeProtocolReadEntryData(t *testing.T) {
	proto := NewLikeProtocol()
	profileData := []byte{
		0, 0, 0, 0, 0, 0, 0, 42, // user id
		0, 0, 0, 2, // like count
		0, 0, 0, 0, 0, 0, 0, 7, // like #1
		0, 0, 0, 0, 0, 0, 0, 13, // like #2
		0, 0, 0, 1, // dislike count
		0, 0, 0, 0, 0, 0, 0, 33, // dislike #1
	}
	profile := entities.Profile{
		UserID:   42,
		Likes:    []uint64{7, 13},
		Dislikes: []uint64{33},
	}

	t.Run("should read a profile", func(t *testing.T) {
		entry := &Entry{
			Capacity: uint32(len(profileData) + entryHeaderSize),
		}
		n, err := proto.ReadEntryData(entry, bytes.NewReader(profileData))
		if err != nil {
			t.Error(err)
			return
		}
		if n != len(profileData) {
			t.Errorf("Read %d bytes, must be %d", n, len(profileData))
			return
		}
		resultProfile, ok := entry.Data.(*entities.Profile)
		if !ok {
			t.Errorf("Invalid Data type %s", reflect.TypeOf(resultProfile).Name())
			return
		}
		if !reflect.DeepEqual(*resultProfile, profile) {
			t.Errorf("Expected profile to be %v, got %v", profile, *resultProfile)
			return
		}
	})

	t.Run("should fail reading profile data larger than available capacity", func(t *testing.T) {
		entry := &Entry{
			Capacity: uint32(len(profileData)),
		}
		_, err := proto.ReadEntryData(entry, bytes.NewReader(profileData))
		if err == nil {
			t.Error("Expected an error")
			return
		}
	})
}

func TestLikeProtocolWriteEntryData(t *testing.T) {
	proto := NewLikeProtocol()
	profileData := []byte{
		0, 0, 0, 0, 0, 0, 0, 42, // user id
		0, 0, 0, 2, // like count
		0, 0, 0, 0, 0, 0, 0, 7, // like #1
		0, 0, 0, 0, 0, 0, 0, 13, // like #2
		0, 0, 0, 1, // dislike count
		0, 0, 0, 0, 0, 0, 0, 33, // dislike #1
	}
	profile := entities.Profile{
		UserID:   42,
		Likes:    []uint64{7, 13},
		Dislikes: []uint64{33},
	}

	t.Run("should write a profile", func(t *testing.T) {
		buffer := helpers.NewFileBuffer(nil)
		entry := Entry{
			Capacity: uint32(len(profileData) + entryHeaderSize),
			Deleted:  0,
			Data:     &profile,
		}
		n, err := proto.WriteEntryData(&entry, buffer)
		if err != nil {
			t.Error(err)
			return
		}
		if n != len(profileData) {
			t.Errorf("Expected written len to be %d, got %d", len(profileData), n)
			return
		}
		if !reflect.DeepEqual(profileData, buffer.Bytes()) {
			t.Errorf("Expected buffer to be %v, got %v", profileData, buffer.Bytes())
			return
		}
	})
}
