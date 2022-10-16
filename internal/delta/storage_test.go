package delta

import (
	"recengine/internal/helpers"
	"reflect"
	"testing"
)

func makeTestHeaderData(locked bool, numEntries int) []byte {
	hdr := &header{
		version:    version,
		locked:     0,
		numEntries: uint32(numEntries),
	}
	if locked {
		hdr.locked = 1
	}
	file := helpers.NewFileBuffer(nil)
	defer file.Close()
	writePrefix(file)
	writeHeader(hdr, file)
	return file.Bytes()
}

func makeTestEntryData(op Operation, user uint64, item uint64) []byte {
	dto := &entry{
		op:   op,
		user: user,
		item: item,
	}
	dto.checksum = calcEntryChecksum(dto)
	file := helpers.NewFileBuffer(nil)
	defer file.Close()
	writeEntry(dto, file)
	return file.Bytes()
}

func TestRecover(t *testing.T) {
	t.Run("should recover corrupted files", func(t *testing.T) {
		lockedHeader := makeTestHeaderData(true, 42)
		soleHeader := makeTestHeaderData(false, 1)
		validEntry := makeTestEntryData(OpRemove, 7, 13)
		invalidEntry := makeTestEntryData(OpRemove, 7, 13)
		invalidEntry[len(invalidEntry)-1] = 0 // Checksum
		fileData := append(append(lockedHeader, validEntry...), invalidEntry...)
		expected := append(soleHeader, validEntry...)
		file := helpers.NewFileBuffer(fileData)
		err := Recover(file)
		if err != nil {
			t.Errorf("Got error: %v", err)
			return
		}
		if !reflect.DeepEqual(file.Bytes(), expected) {
			t.Errorf("Expected data \n%v, got \n%v", expected, file.Bytes())
		}
	})
}

func TestOpen(t *testing.T) {
	t.Run("should create a new one if the file is empty", func(t *testing.T) {
		expectedFileData := makeTestHeaderData(true, 0)
		file := helpers.NewFileBuffer(nil)
		storage, err := Open(file)
		if err != nil {
			t.Errorf("Got error: %v", err)
			return
		}
		defer storage.Close()
		if !reflect.DeepEqual(file.Bytes(), expectedFileData) {
			t.Errorf("Expected data \n%v, got \n%v", expectedFileData, file.Bytes())
		}
	})

	t.Run("should open the file if it is not empty", func(t *testing.T) {
		headerData := makeTestHeaderData(false, 1)
		entryData := makeTestEntryData(OpRemove, 7, 13)
		file := helpers.NewFileBuffer(append(headerData, entryData...))
		storage, err := Open(file)
		if err != nil {
			t.Errorf("Got error: %v", err)
			return
		}
		defer storage.Close()
		if storage.GetTotalItemCount() != 1 {
			t.Errorf("total item count expected %d, got %d", 1, storage.GetTotalItemCount())
		}
		if storage.GetUserCount() != 1 {
			t.Errorf("user count expected %d, got %d", 1, storage.GetUserCount())
		}
	})

	t.Run("should fail opening a locked file", func(t *testing.T) {
		headerData := makeTestHeaderData(true, 0)
		file := helpers.NewFileBuffer(headerData)
		storage, err := Open(file)
		if err == nil {
			storage.Close()
			t.Error("Opened a locked file without an error")
		}
	})

	t.Run("should fail opening a malformed file", func(t *testing.T) {
		headerData := makeTestHeaderData(true, 1)
		file := helpers.NewFileBuffer(headerData)
		storage, err := Open(file)
		if err == nil {
			storage.Close()
			t.Error("Opened a malformed file without an error")
		}
	})

	t.Run("the file should stay locked until closed", func(t *testing.T) {
		file := helpers.NewFileBuffer(nil)
		storage, err := Open(file)
		if err != nil {
			t.Errorf("Got error: %v", err)
			return
		}
		defer storage.Close()
		locked, _ := IsLocked(file)
		if !locked {
			t.Error("The file is not locked")
		}
	})
}

func TestOpenMaybeRecover(t *testing.T) {
	t.Run("should recover corrupted files", func(t *testing.T) {
		lockedHeader := makeTestHeaderData(true, 42)
		soleHeader := makeTestHeaderData(false, 1)
		validEntry := makeTestEntryData(OpRemove, 7, 13)
		invalidEntry := makeTestEntryData(OpRemove, 7, 13)
		invalidEntry[len(invalidEntry)-1] = 0 // Checksum
		fileData := append(append(lockedHeader, validEntry...), invalidEntry...)
		expected := append(soleHeader, validEntry...)
		file := helpers.NewFileBuffer(fileData)
		storage, err := OpenMaybeRecover(file)
		if err != nil {
			t.Errorf("Got error: %v", err)
			return
		}
		storage.Close()
		if !reflect.DeepEqual(file.Bytes(), expected) {
			t.Errorf("Expected data \n%v, got \n%v", expected, file.Bytes())
		}
	})
}

func TestClose(t *testing.T) {
	t.Run("the file should be unlocked after closed", func(t *testing.T) {
		file := helpers.NewFileBuffer(nil)
		storage, err := Open(file)
		if err != nil {
			t.Errorf("Got error opening the file: %v", err)
			return
		}
		err = storage.Close()
		if err != nil {
			t.Errorf("Got error closing the file: %v", err)
			return
		}
		locked, _ := IsLocked(file)
		if locked {
			t.Error("The file is locked")
		}
	})

	t.Run("should flush the cache", func(t *testing.T) {
		// Create a file with 3 items
		file := helpers.NewFileBuffer(nil)
		storage, err := Open(file)
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
		storage, err = Open(file)
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
	t.Run("should add entries to local cache", func(t *testing.T) {
		// Add items
		file := helpers.NewFileBuffer(nil)
		storage, err := Open(file)
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
		storage, err := Open(file)
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
		storage, err := Open(file)
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
