package index

import (
	"io"
	"recengine/internal/helpers"
	"testing"
)

func TestOpen(t *testing.T) {
	t.Run("should create a storage that is locked until closed", func(t *testing.T) {
		file := helpers.NewFileBuffer(nil)
		storage, err := Open(file, nil)
		if err != nil {
			t.Errorf("Failed to open: %v", err)
			return
		}
		locked, _ := IsLocked(file)
		if !locked {
			t.Error("The storage is unlocked before closed")
		}
		storage.Close()
		file.Seek(0, io.SeekStart)
		_, err = readPrefix(file)
		if err != nil {
			t.Errorf("Invalid prefix: %v", err)
			return
		}
		header := header{}
		_, err = readHeader(&header, file)
		if err != nil {
			t.Errorf("Invalid header: %v", err)
			return
		}
		if header.version != version {
			t.Errorf("Invalid version, got %d", header.version)
			return
		}
		if header.locked != 0 {
			t.Error("The storage created unlocked")
			return
		}
		if header.numEntries != 0 {
			t.Errorf("Expected to have 0 entries, got %d", header.numEntries)
			return
		}
		if file.Len() != len(prefix)+headerSize {
			t.Errorf("Expected be of %d bytes, got %d", file.Len(), len(prefix)+headerSize)
			return
		}
	})

	t.Run("should open a storage that is locked until closed", func(t *testing.T) {
		file := helpers.NewFileBuffer(nil)
		storage, err := Open(file, nil)
		if err != nil {
			t.Errorf("Failed to open: %v", err)
			return
		}
		storage.Close()
		storage, err = Open(file, nil)
		if err != nil {
			t.Errorf("Failed to open: %v", err)
			return
		}
		locked, _ := IsLocked(file)
		if !locked {
			t.Error("The storage is unlocked before closed")
		}
		storage.Close()
		locked, _ = IsLocked(file)
		if locked {
			t.Error("The storage is locked after closed")
		}
	})
}

func TestPut(t *testing.T) {
	t.Run("should add index into the memory", func(t *testing.T) {
		file := helpers.NewFileBuffer(nil)
		storage, err := Open(file, nil)
		if err != nil {
			t.Errorf("Failed to open: %v", err)
			return
		}
		defer storage.Close()

		_, ok := storage.Get(7)
		if ok {
			t.Error("The index exists before put")
			return
		}
		err = storage.Put(7, 42)
		if err != nil {
			t.Errorf("Failed to put index: %v", err)
			return
		}
		if idx, ok := storage.Get(7); !ok || idx != 42 {
			t.Errorf("Expected index to be 42, got %d (%v)", idx, ok)
			return
		}
	})

	t.Run("should add index into the file", func(t *testing.T) {
		file := helpers.NewFileBuffer(nil)
		storage, err := Open(file, nil)
		if err != nil {
			t.Errorf("Failed to open: %v", err)
			return
		}

		err = storage.Put(7, 42)
		if err != nil {
			t.Errorf("Failed to put index: %v", err)
			return
		}
		err = storage.Put(13, 11)
		if err != nil {
			t.Errorf("Failed to put index: %v", err)
			return
		}

		storage.Close()

		file.Seek(0, io.SeekStart)
		_, err = readPrefix(file)
		if err != nil {
			t.Errorf("Failed to read prefix: %v", err)
			return
		}
		header := header{}
		_, err = readHeader(&header, file)
		if err != nil {
			t.Errorf("Failed to read header: %v", err)
			return
		}
		if header.numEntries != 2 {
			t.Errorf("Expected to have 2 entries, got %d", header.numEntries)
			return
		}

		entry := entry{}
		_, err = readEntry(&entry, file)
		if err != nil {
			t.Errorf("Failed to read entry: %v", err)
			return
		}
		if entry.id != 7 || entry.index != 42 {
			t.Errorf("Expected 1st entry to be 7 42, got %d %d", entry.id, entry.index)
			return
		}

		_, err = readEntry(&entry, file)
		if err != nil {
			t.Errorf("Failed to read entry: %v", err)
			return
		}
		if entry.id != 13 || entry.index != 11 {
			t.Errorf("Expected 1st entry to be 13 11, got %d %d", entry.id, entry.index)
			return
		}
	})
}

func TestRemove(t *testing.T) {
	t.Run("should delete index from memory", func(t *testing.T) {
		file := helpers.NewFileBuffer(nil)
		storage, err := Open(file, nil)
		if err != nil {
			t.Errorf("Failed to open: %v", err)
			return
		}
		defer storage.Close()

		err = storage.Put(7, 42)
		if err != nil {
			t.Errorf("Failed to put index: %v", err)
			return
		}
		err = storage.Put(11, 13)
		if err != nil {
			t.Errorf("Failed to put index: %v", err)
			return
		}
		err = storage.Remove(7)
		if err != nil {
			t.Errorf("Failed to remove index: %v", err)
			return
		}
		if _, ok := storage.Get(7); ok {
			t.Error("The index available after removal")
			return
		}
		if _, ok := storage.Get(11); !ok {
			t.Error("The index is unavailable")
			return
		}
	})

	t.Run("should delete index from the file", func(t *testing.T) {
		file := helpers.NewFileBuffer(nil)
		storage, err := Open(file, nil)
		if err != nil {
			t.Errorf("Failed to open: %v", err)
			return
		}

		err = storage.Put(7, 42)
		if err != nil {
			t.Errorf("Failed to put index: %v", err)
			return
		}
		err = storage.Put(13, 11)
		if err != nil {
			t.Errorf("Failed to put index: %v", err)
			return
		}
		err = storage.Remove(7)
		if err != nil {
			t.Errorf("Failed to remove index: %v", err)
			return
		}

		storage.Close()

		file.Seek(0, io.SeekStart)
		_, err = readPrefix(file)
		if err != nil {
			t.Errorf("Failed to read prefix: %v", err)
			return
		}
		header := header{}
		_, err = readHeader(&header, file)
		if err != nil {
			t.Errorf("Failed to read header: %v", err)
			return
		}
		if header.numEntries != 1 {
			t.Errorf("Expected to have 1 entries, got %d", header.numEntries)
			return
		}

		entry := entry{}
		_, err = readEntry(&entry, file)
		if err != nil {
			t.Errorf("Failed to read entry: %v", err)
			return
		}
		if entry.id != 13 || entry.index != 11 {
			t.Errorf("Expected 1st entry to be 0 13 11, got %d %d", entry.id, entry.index)
			return
		}
	})
}
