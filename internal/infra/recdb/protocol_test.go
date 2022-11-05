package recdb

import (
	"bytes"
	"io"
	"recengine/internal/domain/entities"
	"recengine/internal/helpers"
	"reflect"
	"testing"
)

func TestProtocolWritePrefix(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	proto := NewProtocol(NewLikeProtocol())
	n, err := proto.WritePrefix(buf)
	if err != nil {
		t.Errorf("Got error: %v", err)
		return
	}
	if n != len(prefix) {
		t.Errorf("Write length expected %d, got %d", len(prefix), n)
		return
	}
	if !reflect.DeepEqual(buf.Bytes(), prefix[:]) {
		t.Errorf("Prefix expected %v, got %v", prefix, buf.Bytes())
		return
	}
}

func TestProtocolReadPrefix(t *testing.T) {
	reader := bytes.NewReader(append(prefix[:], 42))
	proto := NewProtocol(NewLikeProtocol())
	n, err := proto.ReadPrefix(reader)
	if err != nil {
		t.Errorf("Got error: %v", err)
		return
	}
	if n != len(prefix) {
		t.Errorf("Read length expected %d, got %d", len(prefix), n)
		return
	}
	pos, _ := reader.Seek(0, io.SeekCurrent)
	if pos != int64(len(prefix)) {
		t.Errorf("Position after read expected %d, got %d", len(prefix), pos)
		return
	}
}

func TestProtocolWriteHeader(t *testing.T) {
	header := Header{1, [...]byte{'L', 'I', 'K', 'E', ' ', ' ', ' ', ' '}, 0, 42}
	expected := []byte{1, 'L', 'I', 'K', 'E', ' ', ' ', ' ', ' ', 0, 0, 0, 0, 42}
	buf := bytes.NewBuffer(nil)
	proto := NewProtocol(NewLikeProtocol())
	n, err := proto.WriteHeader(&header, buf)
	if err != nil {
		t.Errorf("Got error: %v", err)
		return
	}
	if n != len(expected) {
		t.Errorf("Write length expected %d, got %d", len(expected), n)
		return
	}
	if !reflect.DeepEqual(expected, buf.Bytes()) {
		t.Errorf("Header expected %v, got %v", expected, buf.Bytes())
		return
	}
}

func TestProtocolReadHeader(t *testing.T) {
	data := []byte{2, 'L', 'I', 'K', 'E', ' ', ' ', ' ', ' ', 1, 0, 0, 0, 42}
	expected := Header{2, [...]byte{'L', 'I', 'K', 'E', ' ', ' ', ' ', ' '}, 1, 42}
	header := Header{}
	reader := bytes.NewReader(append(data, 7))
	proto := NewProtocol(NewLikeProtocol())
	n, err := proto.ReadHeader(&header, reader)
	if err != nil {
		t.Errorf("Got error: %v", err)
		return
	}
	if n != len(data) {
		t.Errorf("Read length expected %d, got %d", len(data), n)
		return
	}
	pos, _ := reader.Seek(0, io.SeekCurrent)
	if pos != int64(len(data)) {
		t.Errorf("Position after read expected %d, got %d", len(data), pos)
		return
	}
	if !reflect.DeepEqual(expected, header) {
		t.Errorf("Header expected %v, got %v", expected, header)
		return
	}
}

func TestProtocolWriteEntry(t *testing.T) {
	profile := entities.NewProfile(42)
	profile.Likes = []uint64{7, 13}
	profile.Dislikes = []uint64{33}
	expected := []byte{
		0, 0, 0, 50, // Capacity
		0,                       // Deleted
		0, 0, 0, 0, 0, 0, 0, 42, // user id
		0, 0, 0, 2, // like count
		0, 0, 0, 0, 0, 0, 0, 7, // like #1
		0, 0, 0, 0, 0, 0, 0, 13, // like #2
		0, 0, 0, 1, // dislike count
		0, 0, 0, 0, 0, 0, 0, 33, // dislike #1
		0, 0, 0, 0, 0, // reserve
	}
	entry := Entry{uint32(len(expected)), 0, profile}
	buf := bytes.NewBuffer(nil)
	proto := NewProtocol(NewLikeProtocol())
	n, err := proto.WriteEntry(&entry, buf)
	if err != nil {
		t.Errorf("Got error: %v", err)
		return
	}
	if n != len(expected) {
		t.Errorf("Write length expected %d, got %d", len(expected), n)
		return
	}
	if !reflect.DeepEqual(expected, buf.Bytes()) {
		t.Errorf("Entry expected \n%v, got \n%v", expected, buf.Bytes())
		return
	}
}

func TestReadEntry(t *testing.T) {
	data := []byte{
		0, 0, 0, 50, // Capacity
		0,                       // Deleted
		0, 0, 0, 0, 0, 0, 0, 42, // user id
		0, 0, 0, 2, // like count
		0, 0, 0, 0, 0, 0, 0, 7, // like #1
		0, 0, 0, 0, 0, 0, 0, 13, // like #2
		0, 0, 0, 1, // dislike count
		0, 0, 0, 0, 0, 0, 0, 33, // dislike #1
		0, 0, 0, 0, 0, // reserve
	}
	profile := entities.NewProfile(42)
	profile.Likes = []uint64{7, 13}
	profile.Dislikes = []uint64{33}
	expected := Entry{uint32(len(data)), 0, profile}
	entry := Entry{}
	reader := bytes.NewReader(append(data, 42))
	proto := NewProtocol(NewLikeProtocol())
	n, err := proto.ReadEntry(&entry, reader)
	if err != nil {
		t.Errorf("Got error: %v", err)
		return
	}
	if n != len(data) {
		t.Errorf("Read length expected %d, got %d", len(data), n)
		return
	}
	pos, _ := reader.Seek(0, io.SeekCurrent)
	if pos != int64(len(data)) {
		t.Errorf("Position after read expected %d, got %d", len(data), pos)
		return
	}
	if !reflect.DeepEqual(expected, entry) {
		t.Errorf("Header expected %v, got %v", expected, entry)
		return
	}
}

func TestWriteLocked(t *testing.T) {
	unlockedHeader := append(
		prefix[:],
		1,                                      // Version
		'L', 'I', 'K', 'E', ' ', ' ', ' ', ' ', // EntryType
		0,           // Locked
		0, 0, 0, 42, // NumEntries
	)
	lockedHeader := append(
		prefix[:],
		1,                                      // Version
		'L', 'I', 'K', 'E', ' ', ' ', ' ', ' ', // EntryType
		1,           // Locked
		0, 0, 0, 42, // NumEntries
	)
	t.Run("should lock a file", func(t *testing.T) {
		buf := helpers.NewFileBuffer(append([]byte{}, unlockedHeader...))
		proto := NewProtocol(NewLikeProtocol())
		err := proto.WriteLocked(true, buf)
		if err != nil {
			t.Errorf("Got error: %v", err)
			return
		}
		if !reflect.DeepEqual(lockedHeader, buf.Bytes()) {
			t.Errorf("Header expected \n%v, got \n%v", lockedHeader, buf.Bytes())
			return
		}
	})
	t.Run("should unlock a file", func(t *testing.T) {
		buf := helpers.NewFileBuffer(append([]byte{}, lockedHeader...))
		proto := NewProtocol(NewLikeProtocol())
		err := proto.WriteLocked(false, buf)
		if err != nil {
			t.Errorf("Got error: %v", err)
			return
		}
		if !reflect.DeepEqual(unlockedHeader, buf.Bytes()) {
			t.Errorf("Header expected \n%v, got \n%v", unlockedHeader, buf.Bytes())
			return
		}
	})
}

func TestIsLocked(t *testing.T) {
	unlockedHeader := append(
		prefix[:],
		1,                                      // Version
		'L', 'I', 'K', 'E', ' ', ' ', ' ', ' ', // EntryType
		0,           // Locked
		0, 0, 0, 42, // NumEntries
	)
	lockedHeader := append(
		prefix[:],
		1,                                      // Version
		'L', 'I', 'K', 'E', ' ', ' ', ' ', ' ', // EntryType
		1,           // Locked
		0, 0, 0, 42, // NumEntries
	)
	t.Run("should return true for locked file", func(t *testing.T) {
		buf := helpers.NewFileBuffer(lockedHeader)
		proto := NewProtocol(NewLikeProtocol())
		locked, err := proto.IsLocked(buf)
		if err != nil {
			t.Errorf("Got error: %v", err)
			return
		}
		if !locked {
			t.Error("False negative")
			return
		}
	})
	t.Run("should return false for unlocked file", func(t *testing.T) {
		buf := helpers.NewFileBuffer(unlockedHeader)
		proto := NewProtocol(NewLikeProtocol())
		locked, err := proto.IsLocked(buf)
		if err != nil {
			t.Errorf("Got error: %v", err)
			return
		}
		if locked {
			t.Error("False positive")
			return
		}
	})
}
