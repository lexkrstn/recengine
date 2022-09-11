package index

import (
	"bytes"
	"io"
	"recengine/internal/helpers"
	"reflect"
	"testing"
)

func TestWritePrefix(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	n, err := writePrefix(buf)
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

func TestReadPrefix(t *testing.T) {
	reader := bytes.NewReader(append(prefix[:], 42))
	n, err := readPrefix(reader)
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

func TestWriteHeader(t *testing.T) {
	header := header{1, 0, 42}
	expected := []byte{1, 0, 0, 0, 0, 42}
	buf := bytes.NewBuffer(nil)
	n, err := writeHeader(&header, buf)
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

func TestReadHeader(t *testing.T) {
	data := []byte{2, 1, 0, 0, 0, 42}
	expected := header{2, 1, 42}
	header := header{}
	reader := bytes.NewReader(append(data, 7))
	n, err := readHeader(&header, reader)
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

func TestWriteEntry(t *testing.T) {
	entry := entry{1, 7, 13}
	expected := []byte{1, 0, 0, 0, 0, 0, 0, 0, 7, 0, 0, 0, 0, 0, 0, 0, 13}
	buf := bytes.NewBuffer(nil)
	n, err := writeEntry(&entry, buf)
	if err != nil {
		t.Errorf("Got error: %v", err)
		return
	}
	if n != len(expected) {
		t.Errorf("Write length expected %d, got %d", len(expected), n)
		return
	}
	if !reflect.DeepEqual(expected, buf.Bytes()) {
		t.Errorf("Entry expected %v, got %v", expected, buf.Bytes())
		return
	}
}

func TestReadEntry(t *testing.T) {
	data := []byte{1, 0, 0, 0, 0, 0, 0, 0, 7, 0, 0, 0, 0, 0, 0, 0, 13}
	expected := entry{1, 7, 13}
	entry := entry{}
	reader := bytes.NewReader(append(data, 42))
	n, err := readEntry(&entry, reader)
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

func TestWriteEntryDeleted(t *testing.T) {
	expected := []byte{1}
	buf := bytes.NewBuffer(nil)
	err := writeEntryDeleted(true, buf)
	if err != nil {
		t.Errorf("Got error: %v", err)
		return
	}
	if !reflect.DeepEqual(expected, buf.Bytes()) {
		t.Errorf("Entry expected %v, got %v", expected, buf.Bytes())
		return
	}
}

func TestWriteLocked(t *testing.T) {
	unlockedHeader := append(prefix[:], 1, 0, 0, 0, 0, 42)
	lockedHeader := append(prefix[:], 1, 1, 0, 0, 0, 42)
	t.Run("should lock a file", func(t *testing.T) {
		buf := helpers.NewFileBuffer(append([]byte{}, unlockedHeader...))
		err := WriteLocked(true, &buf)
		if err != nil {
			t.Errorf("Got error: %v", err)
			return
		}
		if !reflect.DeepEqual(lockedHeader, buf.Bytes()) {
			t.Errorf("Header expected %v, got %v", lockedHeader, buf.Bytes())
			return
		}
	})
	t.Run("should unlock a file", func(t *testing.T) {
		buf := helpers.NewFileBuffer(append([]byte{}, lockedHeader...))
		err := WriteLocked(false, &buf)
		if err != nil {
			t.Errorf("Got error: %v", err)
			return
		}
		if !reflect.DeepEqual(unlockedHeader, buf.Bytes()) {
			t.Errorf("Header expected %v, got %v", unlockedHeader, buf.Bytes())
			return
		}
	})
}

func TestIsLocked(t *testing.T) {
	unlockedHeader := append(prefix[:], 1, 0, 0, 0, 0, 42)
	lockedHeader := append(prefix[:], 1, 1, 0, 0, 0, 42)
	t.Run("should return true for locked file", func(t *testing.T) {
		buf := helpers.NewFileBuffer(lockedHeader)
		locked, err := IsLocked(&buf)
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
		locked, err := IsLocked(&buf)
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
