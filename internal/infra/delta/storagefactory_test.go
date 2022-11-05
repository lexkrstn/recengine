package delta

import (
	"recengine/internal/helpers"
	"reflect"
	"testing"
)

func TestRecover(t *testing.T) {
	factory := NewFactory()

	t.Run("should recover corrupted files", func(t *testing.T) {
		lockedHeader := makeTestHeaderData(true, 42)
		soleHeader := makeTestHeaderData(false, 1)
		validEntry := makeTestEntryData(OpRemove, 7, 13)
		invalidEntry := makeTestEntryData(OpRemove, 7, 13)
		invalidEntry[len(invalidEntry)-1] = 0 // Checksum
		fileData := append(append(lockedHeader, validEntry...), invalidEntry...)
		expected := append(soleHeader, validEntry...)
		file := helpers.NewFileBuffer(fileData)
		err := factory.Recover(file)
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
	factory := NewFactory()

	t.Run("should create a new one if the file is empty", func(t *testing.T) {
		expectedFileData := makeTestHeaderData(true, 0)
		file := helpers.NewFileBuffer(nil)
		storage, err := factory.Open(file)
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
		storage, err := factory.Open(file)
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
		storage, err := factory.Open(file)
		if err == nil {
			storage.Close()
			t.Error("Opened a locked file without an error")
		}
	})

	t.Run("should fail opening a malformed file", func(t *testing.T) {
		headerData := makeTestHeaderData(true, 1)
		file := helpers.NewFileBuffer(headerData)
		storage, err := factory.Open(file)
		if err == nil {
			storage.Close()
			t.Error("Opened a malformed file without an error")
		}
	})

	t.Run("the file should stay locked until closed", func(t *testing.T) {
		file := helpers.NewFileBuffer(nil)
		storage, err := factory.Open(file)
		if err != nil {
			t.Errorf("Got error: %v", err)
			return
		}
		defer storage.Close()
		deltaFile := &Protocol{}
		locked, _ := deltaFile.IsLocked(file)
		if !locked {
			t.Error("The file is not locked")
		}
	})
}

func TestOpenMaybeRecover(t *testing.T) {
	factory := NewFactory()

	t.Run("should recover corrupted files", func(t *testing.T) {
		lockedHeader := makeTestHeaderData(true, 42)
		soleHeader := makeTestHeaderData(false, 1)
		validEntry := makeTestEntryData(OpRemove, 7, 13)
		invalidEntry := makeTestEntryData(OpRemove, 7, 13)
		invalidEntry[len(invalidEntry)-1] = 0 // Checksum
		fileData := append(append(lockedHeader, validEntry...), invalidEntry...)
		expected := append(soleHeader, validEntry...)
		file := helpers.NewFileBuffer(fileData)
		storage, err := factory.OpenMaybeRecover(file)
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
