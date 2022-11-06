package recdb

import (
	"errors"
	"recengine/internal/domain"
	"recengine/internal/helpers"
	"reflect"
	"testing"
)

func mockLikeRecDbHeaderBytes(locked bool, numEntries byte) []byte {
	lockedByte := byte(0)
	if locked {
		lockedByte = 1
	}
	return append(
		prefix[:],
		1,                                      // Version
		'L', 'I', 'K', 'E', ' ', ' ', ' ', ' ', // EntryType
		lockedByte,          // Locked
		0, 0, 0, numEntries, // NumEntries
	)
}

func mockLikeRecDbEntryBytes(deleted bool) []byte {
	deletedByte := byte(0)
	if deleted {
		deletedByte = 1
	}
	return []byte{
		0, 0, 0, 50, // Capacity
		deletedByte,             // Deleted
		0, 0, 0, 0, 0, 0, 0, 42, // user id
		0, 0, 0, 2, // like count
		0, 0, 0, 0, 0, 0, 0, 7, // like #1
		0, 0, 0, 0, 0, 0, 0, 13, // like #2
		0, 0, 0, 1, // dislike count
		0, 0, 0, 0, 0, 0, 0, 33, // dislike #1
		0, 0, 0, 0, 0, // reserve
	}
}

func mockLikeRecDbEntry(deleted bool) *Entry {
	deletedByte := byte(0)
	if deleted {
		deletedByte = 1
	}
	return &Entry{
		Capacity: 50,
		Deleted:  deletedByte,
		Data: &domain.Profile{
			UserID:   42,
			Likes:    []uint64{7, 13},
			Dislikes: []uint64{33},
		},
	}
}

func TestNewIterator(t *testing.T) {
	t.Run("should open valid file", func(t *testing.T) {
		buffer := helpers.NewFileBuffer(mockLikeRecDbHeaderBytes(false, 0))
		_, err := NewIterator(buffer, NewProtocol(NewLikeProtocol()))
		if err != nil {
			t.Error(err)
			return
		}
	})

	t.Run("should fail opening invalid file", func(t *testing.T) {
		buffer := helpers.NewFileBuffer([]byte{1, 2, 3, 4, 5})
		_, err := NewIterator(buffer, NewProtocol(NewLikeProtocol()))
		if err == nil {
			t.Error("expected an error")
			return
		}
	})

	t.Run("should fail opening locked file", func(t *testing.T) {
		buffer := helpers.NewFileBuffer(mockLikeRecDbHeaderBytes(true, 0))
		_, err := NewIterator(buffer, NewProtocol(NewLikeProtocol()))
		if err == nil || !errors.Is(err, &domain.CorruptedFileError{}) {
			t.Error("expected an error")
			return
		}
	})
}

func TestIteratorHasNext(t *testing.T) {
	t.Run("should return false if there is no entries", func(t *testing.T) {
		data := mockLikeRecDbHeaderBytes(false, 0)
		buffer := helpers.NewFileBuffer(data)
		iter, err := NewIterator(buffer, NewProtocol(NewLikeProtocol()))
		if err != nil {
			t.Error(err)
			return
		}
		if iter.HasNext() {
			t.Error("expected HasNext() to be false")
		}
	})
	t.Run("should return true if there is an entry", func(t *testing.T) {
		data := mockLikeRecDbHeaderBytes(false, 1)
		buffer := helpers.NewFileBuffer(data)
		iter, err := NewIterator(buffer, NewProtocol(NewLikeProtocol()))
		if err != nil {
			t.Error(err)
			return
		}
		if !iter.HasNext() {
			t.Error("expected HasNext() to be true")
		}
	})
	t.Run("should return false if all entries having been read", func(t *testing.T) {
		data := append(
			mockLikeRecDbHeaderBytes(false, 1),
			mockLikeRecDbEntryBytes(false)...,
		)
		buffer := helpers.NewFileBuffer(data)
		iter, err := NewIterator(buffer, NewProtocol(NewLikeProtocol()))
		if err != nil {
			t.Error(err)
			return
		}
		if !iter.HasNext() {
			t.Error("expected HasNext() before Next() to be true")
		}
		iter.Next()
		if iter.HasNext() {
			t.Error("expected HasNext() after Next() to be false")
		}
	})
	t.Run("should return true if next entry marked deleted", func(t *testing.T) {
		data := append(
			mockLikeRecDbHeaderBytes(false, 1),
			mockLikeRecDbEntryBytes(true)...,
		)
		buffer := helpers.NewFileBuffer(data)
		iter, err := NewIterator(buffer, NewProtocol(NewLikeProtocol()))
		if err != nil {
			t.Error(err)
			return
		}
		if !iter.HasNext() {
			t.Error("expected HasNext() before Next() to be true")
		}
	})
}

func TestIteratorNext(t *testing.T) {
	t.Run("should return an error if the file is empty", func(t *testing.T) {
		data := mockLikeRecDbHeaderBytes(false, 0)
		buffer := helpers.NewFileBuffer(data)
		iter, err := NewIterator(buffer, NewProtocol(NewLikeProtocol()))
		if err != nil {
			t.Error(err)
			return
		}
		_, err = iter.Next()
		if err == nil {
			t.Error("expected an error")
			return
		}
	})
	t.Run("should read an entry", func(t *testing.T) {
		data := append(
			mockLikeRecDbHeaderBytes(false, 1),
			mockLikeRecDbEntryBytes(true)...,
		)
		buffer := helpers.NewFileBuffer(data)
		iter, err := NewIterator(buffer, NewProtocol(NewLikeProtocol()))
		if err != nil {
			t.Error(err)
			return
		}
		entry, err := iter.Next()
		if err != nil {
			t.Error(err)
			return
		}
		if !reflect.DeepEqual(entry, mockLikeRecDbEntry(true)) {
			t.Errorf("expected entry %v, got %v", *mockLikeRecDbEntry(true), *entry)
		}
	})
	t.Run("should fail if no entries left", func(t *testing.T) {
		data := append(
			mockLikeRecDbHeaderBytes(false, 1),
			mockLikeRecDbEntryBytes(true)...,
		)
		buffer := helpers.NewFileBuffer(data)
		iter, err := NewIterator(buffer, NewProtocol(NewLikeProtocol()))
		if err != nil {
			t.Error(err)
			return
		}
		_, err = iter.Next()
		if err != nil {
			t.Error(err)
			return
		}
		_, err = iter.Next()
		if err == nil {
			t.Error("expected an error")
			return
		}
	})
}

func TestIteratorRewind(t *testing.T) {
	t.Run("should return no error if the file is empty", func(t *testing.T) {
		data := append(
			mockLikeRecDbHeaderBytes(false, 1),
			mockLikeRecDbEntryBytes(true)...,
		)
		buffer := helpers.NewFileBuffer(data)
		iter, err := NewIterator(buffer, NewProtocol(NewLikeProtocol()))
		if err != nil {
			t.Error(err)
			return
		}
		err = iter.Rewind()
		if err != nil {
			t.Error(err)
			return
		}
	})
	t.Run("should allow rereading an entry", func(t *testing.T) {
		data := append(
			mockLikeRecDbHeaderBytes(false, 1),
			mockLikeRecDbEntryBytes(true)...,
		)
		buffer := helpers.NewFileBuffer(data)
		iter, err := NewIterator(buffer, NewProtocol(NewLikeProtocol()))
		if err != nil {
			t.Error(err)
			return
		}
		entry1, _ := iter.Next()
		err = iter.Rewind()
		if err != nil {
			t.Error(err)
			return
		}
		entry2, err := iter.Next()
		if err != nil {
			t.Error(err)
			return
		}
		if !reflect.DeepEqual(entry1, entry2) {
			t.Errorf("expected 2nd entry to be %v, got %v", entry1, entry2)
			return
		}
	})
}

func TestIteratorSetPrevious(t *testing.T) {
	t.Run("should return error if there is no previous entry", func(t *testing.T) {
		data := append(
			mockLikeRecDbHeaderBytes(false, 1),
			mockLikeRecDbEntryBytes(true)...,
		)
		buffer := helpers.NewFileBuffer(data)
		iter, err := NewIterator(buffer, NewProtocol(NewLikeProtocol()))
		if err != nil {
			t.Error(err)
			return
		}
		err = iter.SetPrevious(mockLikeRecDbEntry(false))
		if err == nil {
			t.Error("expected an error")
			return
		}
	})
	t.Run("should return error if entry capacities are not equal", func(t *testing.T) {
		data := append(
			mockLikeRecDbHeaderBytes(false, 1),
			mockLikeRecDbEntryBytes(true)...,
		)
		buffer := helpers.NewFileBuffer(data)
		iter, err := NewIterator(buffer, NewProtocol(NewLikeProtocol()))
		if err != nil {
			t.Error(err)
			return
		}
		iter.Next()
		entry := mockLikeRecDbEntry(false)
		entry.Capacity -= 1
		err = iter.SetPrevious(entry)
		if err == nil {
			t.Error("expected an error")
			return
		}
	})
	t.Run("should rewrite previous entry", func(t *testing.T) {
		data := append(
			mockLikeRecDbHeaderBytes(false, 1),
			mockLikeRecDbEntryBytes(true)...,
		)
		buffer := helpers.NewFileBuffer(data)
		iter, err := NewIterator(buffer, NewProtocol(NewLikeProtocol()))
		if err != nil {
			t.Error(err)
			return
		}
		iter.Next()
		entry1 := mockLikeRecDbEntry(false)
		entry1.Data.(*domain.Profile).UserID += 1
		err = iter.SetPrevious(entry1)
		if err != nil {
			t.Error(err)
			return
		}
		err = iter.Rewind()
		if err != nil {
			t.Error(err)
			return
		}
		entry2, err := iter.Next()
		if err != nil {
			t.Error(err)
			return
		}
		if !reflect.DeepEqual(entry1, entry2) {
			t.Errorf("expected %v, got %v", entry1, entry2)
			return
		}
	})
}
