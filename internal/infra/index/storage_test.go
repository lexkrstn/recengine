package index

import (
	"io"
	"recengine/internal/helpers"
	"testing"
)

func TestPut(t *testing.T) {
	proto := &Protocol{}
	factory := NewFactoryForProtocol(proto)

	t.Run("should add index into the memory", func(t *testing.T) {
		file := helpers.NewFileBuffer(nil)
		storage, err := factory.Open(file, nil)
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
		storage, err := factory.Open(file, nil)
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
		_, err = proto.ReadPrefix(file)
		if err != nil {
			t.Errorf("Failed to read prefix: %v", err)
			return
		}
		header := Header{}
		_, err = proto.ReadHeader(&header, file)
		if err != nil {
			t.Errorf("Failed to read header: %v", err)
			return
		}
		if header.NumEntries != 2 {
			t.Errorf("Expected to have 2 entries, got %d", header.NumEntries)
			return
		}

		entry := Entry{}
		_, err = proto.ReadEntry(&entry, file)
		if err != nil {
			t.Errorf("Failed to read entry: %v", err)
			return
		}
		if entry.ID != 7 || entry.Index != 42 {
			t.Errorf("Expected 1st entry to be 7 42, got %d %d", entry.ID, entry.Index)
			return
		}

		_, err = proto.ReadEntry(&entry, file)
		if err != nil {
			t.Errorf("Failed to read entry: %v", err)
			return
		}
		if entry.ID != 13 || entry.Index != 11 {
			t.Errorf("Expected 1st entry to be 13 11, got %d %d", entry.ID, entry.Index)
			return
		}
	})
}

func TestRemove(t *testing.T) {
	proto := &Protocol{}
	factory := NewFactoryForProtocol(proto)

	t.Run("should delete index from memory", func(t *testing.T) {
		file := helpers.NewFileBuffer(nil)
		storage, err := factory.Open(file, nil)
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
		storage, err := factory.Open(file, nil)
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
		_, err = proto.ReadPrefix(file)
		if err != nil {
			t.Errorf("Failed to read prefix: %v", err)
			return
		}
		header := Header{}
		_, err = proto.ReadHeader(&header, file)
		if err != nil {
			t.Errorf("Failed to read header: %v", err)
			return
		}
		if header.NumEntries != 1 {
			t.Errorf("Expected to have 1 entries, got %d", header.NumEntries)
			return
		}

		entry := Entry{}
		_, err = proto.ReadEntry(&entry, file)
		if err != nil {
			t.Errorf("Failed to read entry: %v", err)
			return
		}
		if entry.ID != 13 || entry.Index != 11 {
			t.Errorf("Expected 1st entry to be 0 13 11, got %d %d", entry.ID, entry.Index)
			return
		}
	})
}
