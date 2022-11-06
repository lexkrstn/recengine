package delta

import (
	"recengine/internal/helpers"
	"testing"
)

func makeTestHeaderData(locked bool, numEntries int) []byte {
	hdr := &Header{
		Version:    Version,
		Locked:     0,
		NumEntries: uint32(numEntries),
	}
	if locked {
		hdr.Locked = 1
	}
	file := helpers.NewFileBuffer(nil)
	defer file.Close()
	deltaFile := NewProtocol()
	deltaFile.WritePrefix(file)
	deltaFile.WriteHeader(hdr, file)
	return file.Bytes()
}

func makeTestEntryData(op Operation, user uint64, item uint64) []byte {
	dto := &Entry{
		Op:     op,
		UserID: user,
		ItemID: item,
	}
	deltaFile := NewProtocol()
	dto.Checksum = deltaFile.CalcEntryChecksum(dto)
	file := helpers.NewFileBuffer(nil)
	defer file.Close()
	deltaFile.WriteEntry(dto, file)
	return file.Bytes()
}

func TestClose(t *testing.T) {
	factory := NewStorageFactory()

	t.Run("the file should be unlocked after closed", func(t *testing.T) {
		file := helpers.NewFileBuffer(nil)
		storage, err := factory.Open(file)
		if err != nil {
			t.Errorf("Got error opening the file: %v", err)
			return
		}
		err = storage.Close()
		if err != nil {
			t.Errorf("Got error closing the file: %v", err)
			return
		}
		deltaFile := NewProtocol()
		locked, _ := deltaFile.IsLocked(file)
		if locked {
			t.Error("The file is locked")
		}
	})

	t.Run("should flush the cache", func(t *testing.T) {
		// Create a file with 3 items
		file := helpers.NewFileBuffer(nil)
		storage, err := factory.Open(file)
		if err != nil {
			t.Errorf("Got error creating the file: %v", err)
			return
		}
		storage.Add(OpAdd, 7, 13)
		storage.Add(OpRemove, 7, 42)
		storage.Add(OpAdd, 5, 42)
		// Close
		storage.Close()
		// Open the file again
		file = helpers.NewFileBuffer(file.Bytes())
		storage, err = factory.Open(file)
		if err != nil {
			t.Errorf("Got error opening the file: %v", err)
			return
		}
		if storage.GetTotalItemCount() != 3 {
			t.Errorf("total item count expected %d, got %d", 3, storage.GetTotalItemCount())
		}
		if storage.GetUserCount() != 2 {
			t.Errorf("user count expected %d, got %d", 2, storage.GetUserCount())
		}
		op, exists := storage.Get(7, 13)
		if !exists || op != OpAdd {
			t.Errorf("Item {user: 7, item: 13} doesn't exist: %v, %v", op, exists)
		}
		op, exists = storage.Get(7, 42)
		if !exists || op != OpRemove {
			t.Errorf("Item {user: 7, item: 42} doesn't exist: %v, %v", op, exists)
		}
		op, exists = storage.Get(5, 42)
		if !exists || op != OpAdd {
			t.Errorf("Item {user: 5, item: 42} doesn't exist: %v, %v", op, exists)
		}
	})
}

func TestAdd(t *testing.T) {
	factory := NewStorageFactory()

	t.Run("should add entries to local cache", func(t *testing.T) {
		// Add items
		file := helpers.NewFileBuffer(nil)
		storage, err := factory.Open(file)
		if err != nil {
			t.Errorf("Got error creating the file: %v", err)
			return
		}
		defer storage.Close()
		storage.Add(OpAdd, 7, 13)
		storage.Add(OpRemove, 7, 42)
		storage.Add(OpAdd, 5, 42)
		// Check existence
		if storage.GetTotalItemCount() != 3 {
			t.Errorf("total item count expected %d, got %d", 3, storage.GetTotalItemCount())
		}
		if storage.GetUserCount() != 2 {
			t.Errorf("user count expected %d, got %d", 2, storage.GetUserCount())
		}
		op, exists := storage.Get(7, 13)
		if !exists || op != OpAdd {
			t.Errorf("Item {user: 7, item: 13} doesn't exist: %v, %v", op, exists)
		}
		op, exists = storage.Get(7, 42)
		if !exists || op != OpRemove {
			t.Errorf("Item {user: 7, item: 42} doesn't exist: %v, %v", op, exists)
		}
		op, exists = storage.Get(5, 42)
		if !exists || op != OpAdd {
			t.Errorf("Item {user: 5, item: 42} doesn't exist: %v, %v", op, exists)
		}
	})

	t.Run("shouldn't add duplicate entries", func(t *testing.T) {
		// Add items
		file := helpers.NewFileBuffer(nil)
		storage, err := factory.Open(file)
		if err != nil {
			t.Errorf("Got error creating the file: %v", err)
			return
		}
		defer storage.Close()
		storage.Add(OpAdd, 7, 13)
		storage.Add(OpAdd, 7, 13)
		// Check
		if storage.GetTotalItemCount() != 1 {
			t.Errorf("total item count expected %d, got %d", 1, storage.GetTotalItemCount())
		}
		if storage.GetUserCount() != 1 {
			t.Errorf("user count expected %d, got %d", 1, storage.GetUserCount())
		}
	})

	t.Run("should remove opposite entries", func(t *testing.T) {
		// Add items
		file := helpers.NewFileBuffer(nil)
		storage, err := factory.Open(file)
		if err != nil {
			t.Errorf("Got error creating the file: %v", err)
			return
		}
		defer storage.Close()
		storage.Add(OpAdd, 42, 13)
		storage.Add(OpAdd, 7, 13)
		storage.Add(OpRemove, 7, 13)
		// Check
		if storage.GetTotalItemCount() != 2 {
			t.Errorf("total item count expected %d, got %d", 1, storage.GetTotalItemCount())
		}
		op, exists := storage.Get(7, 13)
		if !exists || op != OpRemove {
			t.Errorf("Item {user: 7, item: 13} doesn't exist: %v, %v", op, exists)
		}
	})
}
