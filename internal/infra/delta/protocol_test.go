package delta

import (
	"bytes"
	"io"
	"recengine/internal/helpers"
	"reflect"
	"testing"
)

func TestWritePrefix(t *testing.T) {
	proto := &Protocol{}
	buf := bytes.NewBuffer(nil)
	err := proto.WritePrefix(buf)
	if err != nil {
		t.Errorf("Got error: %v", err)
		return
	}
	if !reflect.DeepEqual(buf.Bytes(), prefix[:]) {
		t.Errorf("Prefix expected %v, got %v", prefix, buf.Bytes())
		return
	}
}

func TestReadPrefix(t *testing.T) {
	proto := &Protocol{}
	reader := bytes.NewReader(append(prefix[:], 42))
	err := proto.ReadPrefix(reader)
	if err != nil {
		t.Errorf("Got error: %v", err)
		return
	}
	pos, _ := reader.Seek(0, io.SeekCurrent)
	if pos != int64(len(prefix)) {
		t.Errorf("Position after read expected %d, got %d", len(prefix), pos)
		return
	}
}

func TestWriteHeader(t *testing.T) {
	proto := &Protocol{}
	header := Header{1, 0, 42}
	expected := []byte{1, 0, 0, 0, 0, 42}
	buf := bytes.NewBuffer(nil)
	err := proto.WriteHeader(&header, buf)
	if err != nil {
		t.Errorf("Got error: %v", err)
		return
	}
	if !reflect.DeepEqual(expected, buf.Bytes()) {
		t.Errorf("Header expected %v, got %v", expected, buf.Bytes())
		return
	}
}

func TestReadHeader(t *testing.T) {
	proto := &Protocol{}
	data := []byte{2, 1, 0, 0, 0, 42}
	expected := Header{2, 1, 42}
	header := Header{}
	reader := bytes.NewReader(append(data, 7))
	err := proto.ReadHeader(&header, reader)
	if err != nil {
		t.Errorf("Got error: %v", err)
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
	proto := &Protocol{}
	entry := Entry{'-', 7, 13, 65}
	expected := []byte{'-', 0, 0, 0, 0, 0, 0, 0, 7, 0, 0, 0, 0, 0, 0, 0, 13, 65}
	buf := bytes.NewBuffer(nil)
	err := proto.WriteEntry(&entry, buf)
	if err != nil {
		t.Errorf("Got error: %v", err)
		return
	}
	if !reflect.DeepEqual(expected, buf.Bytes()) {
		t.Errorf("Entry expected %v, got %v", expected, buf.Bytes())
		return
	}
}

func TestReadEntry(t *testing.T) {
	proto := &Protocol{}
	data := []byte{'+', 0, 0, 0, 0, 0, 0, 0, 7, 0, 0, 0, 0, 0, 0, 0, 13, 65}
	expected := Entry{'+', 7, 13, 65}
	entry := Entry{}
	reader := bytes.NewReader(append(data, 42))
	err := proto.ReadEntry(&entry, reader)
	if err != nil {
		t.Errorf("Got error: %v", err)
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
	proto := &Protocol{}
	unlockedHeader := append(prefix[:], 1, 0, 0, 0, 0, 42)
	lockedHeader := append(prefix[:], 1, 1, 0, 0, 0, 42)
	t.Run("should lock a file", func(t *testing.T) {
		buf := helpers.NewFileBuffer(append([]byte{}, unlockedHeader...))
		err := proto.WriteLocked(true, buf)
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
		proto := &Protocol{}
		buf := helpers.NewFileBuffer(append([]byte{}, lockedHeader...))
		err := proto.WriteLocked(false, buf)
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
	proto := &Protocol{}
	unlockedHeader := append(prefix[:], 1, 0, 0, 0, 0, 42)
	lockedHeader := append(prefix[:], 1, 1, 0, 0, 0, 42)
	t.Run("should return true for locked file", func(t *testing.T) {
		buf := helpers.NewFileBuffer(lockedHeader)
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
		proto := &Protocol{}
		buf := helpers.NewFileBuffer(unlockedHeader)
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

func TestRecoverTo(t *testing.T) {
	lockedHeader := append(prefix[:], 1, 1, 0, 0, 0, 42)
	emptyHeader := append(prefix[:], 1, 0, 0, 0, 0, 0)
	soleHeader := append(prefix[:], 1, 0, 0, 0, 0, 1)
	validEntry := []byte{'-', 0, 0, 0, 0, 0, 0, 0, 7, 0, 0, 0, 0, 0, 0, 0, 13, 65}
	invalidEntry := []byte{'-', 0, 0, 0, 0, 0, 0, 0, 7, 0, 0, 0, 0, 0, 0, 0, 13, 0}
	halfEntry := []byte{'-', 0, 0, 0, 0, 0, 0, 0, 7, 0, 0}

	t.Run("should recover from unexpected EOF in prefix", func(t *testing.T) {
		proto := &Protocol{}
		srcBuf := helpers.NewFileBuffer(prefix[:len(prefix)-2])
		dstBuf := helpers.NewFileBuffer(nil)
		err := proto.RecoverTo(srcBuf, dstBuf)
		if err != nil {
			t.Errorf("Got error: %v", err)
			return
		}
		if dstBuf.Len() != len(emptyHeader) {
			t.Errorf("Length expected %v, got %v", len(emptyHeader), dstBuf.Len())
		}
		if !reflect.DeepEqual(dstBuf.Bytes(), emptyHeader) {
			t.Errorf("Expected data \n%v, got \n%v", emptyHeader, dstBuf.Bytes())
		}
	})

	t.Run("should recover from unexpected EOF in header", func(t *testing.T) {
		proto := &Protocol{}
		srcBuf := helpers.NewFileBuffer(lockedHeader[:len(lockedHeader)-2])
		dstBuf := helpers.NewFileBuffer(nil)
		err := proto.RecoverTo(srcBuf, dstBuf)
		if err != nil {
			t.Errorf("Got error: %v", err)
			return
		}
		if dstBuf.Len() != len(emptyHeader) {
			t.Errorf("Length expected %v, got %v", len(emptyHeader), dstBuf.Len())
		}
		if !reflect.DeepEqual(dstBuf.Bytes(), emptyHeader) {
			t.Errorf("Expected data \n%v, got \n%v", emptyHeader, dstBuf.Bytes())
		}
	})

	t.Run("should recover from unexpected EOF in entry", func(t *testing.T) {
		proto := &Protocol{}
		halfEntryFileData := append(append(lockedHeader, validEntry...), halfEntry...)
		expected := append(soleHeader, validEntry...)
		srcBuf := helpers.NewFileBuffer(halfEntryFileData)
		dstBuf := helpers.NewFileBuffer(nil)
		err := proto.RecoverTo(srcBuf, dstBuf)
		if err != nil {
			t.Errorf("Got error: %v", err)
			return
		}
		if dstBuf.Len() != len(expected) {
			t.Errorf("Length expected %v, got %v", len(expected), dstBuf.Len())
		}
		if !reflect.DeepEqual(dstBuf.Bytes(), expected) {
			t.Errorf("Expected data \n%v, got \n%v", expected, dstBuf.Bytes())
		}
	})

	t.Run("should recover from entry checksum mismatch", func(t *testing.T) {
		proto := &Protocol{}
		invalidEntryFileData := append(append(lockedHeader, validEntry...), invalidEntry...)
		expected := append(soleHeader, validEntry...)
		srcBuf := helpers.NewFileBuffer(invalidEntryFileData)
		dstBuf := helpers.NewFileBuffer(nil)
		err := proto.RecoverTo(srcBuf, dstBuf)
		if err != nil {
			t.Errorf("Got error: %v", err)
			return
		}
		if dstBuf.Len() != len(expected) {
			t.Errorf("Length expected %v, got %v", len(expected), dstBuf.Len())
		}
		if !reflect.DeepEqual(dstBuf.Bytes(), expected) {
			t.Errorf("Expected data \n%v, got \n%v", expected, dstBuf.Bytes())
		}
	})

	t.Run("should recover from entry count mismatch", func(t *testing.T) {
		proto := &Protocol{}
		fileData := append(emptyHeader, validEntry...)
		expected := append(soleHeader, validEntry...)
		srcBuf := helpers.NewFileBuffer(fileData)
		dstBuf := helpers.NewFileBuffer(nil)
		err := proto.RecoverTo(srcBuf, dstBuf)
		if err != nil {
			t.Errorf("Got error: %v", err)
			return
		}
		if dstBuf.Len() != len(expected) {
			t.Errorf("Length expected %v, got %v", len(expected), dstBuf.Len())
		}
		if !reflect.DeepEqual(dstBuf.Bytes(), expected) {
			t.Errorf("Expected data \n%v, got \n%v", expected, dstBuf.Bytes())
		}
	})
}
