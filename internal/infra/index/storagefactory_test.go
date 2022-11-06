package index

import (
	"io"
	"recengine/internal/helpers"
	"testing"
)

func TestOpen(t *testing.T) {
	proto := NewProtocol()

	t.Run("should create a storage that is locked until closed", func(t *testing.T) {
		file := helpers.NewFileBuffer(nil)
		factory := NewStorageFactoryForProtocol(proto)
		storage, err := factory.Open(file, nil)
		if err != nil {
			t.Errorf("Failed to open: %v", err)
			return
		}
		locked, _ := proto.IsLocked(file)
		if !locked {
			t.Error("The storage is unlocked before closed")
		}
		storage.Close()
		file.Seek(0, io.SeekStart)
		_, err = proto.ReadPrefix(file)
		if err != nil {
			t.Errorf("Invalid prefix: %v", err)
			return
		}
		header := Header{}
		_, err = proto.ReadHeader(&header, file)
		if err != nil {
			t.Errorf("Invalid header: %v", err)
			return
		}
		if header.Version != Version {
			t.Errorf("Invalid version, got %d", header.Version)
			return
		}
		if header.Locked != 0 {
			t.Error("The storage created unlocked")
			return
		}
		if header.NumEntries != 0 {
			t.Errorf("Expected to have 0 entries, got %d", header.NumEntries)
			return
		}
		if file.Len() != len(prefix)+headerSize {
			t.Errorf("Expected be of %d bytes, got %d", file.Len(), len(prefix)+headerSize)
			return
		}
	})

	t.Run("should open a storage that is locked until closed", func(t *testing.T) {
		file := helpers.NewFileBuffer(nil)
		factory := NewStorageFactoryForProtocol(proto)
		storage, err := factory.Open(file, nil)
		if err != nil {
			t.Errorf("Failed to open: %v", err)
			return
		}
		storage.Close()
		storage, err = factory.Open(file, nil)
		if err != nil {
			t.Errorf("Failed to open: %v", err)
			return
		}
		locked, _ := proto.IsLocked(file)
		if !locked {
			t.Error("The storage is unlocked before closed")
		}
		storage.Close()
		locked, _ = proto.IsLocked(file)
		if locked {
			t.Error("The storage is locked after closed")
		}
	})
}
