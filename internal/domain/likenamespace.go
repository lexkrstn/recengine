package domain

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"recengine/internal/domain/valueobjects"
	"time"
)

// likeNamespace performs the same function as databases in relational databases.
type likeNamespace struct {
	name                    valueobjects.NamespaceName
	maxSimilarProfiles      uint
	dislikeFactor           float32
	actionQueueFillWaitTime time.Duration
	basePath                string
	deltaStorageFactory     DeltaStorageFactory
	likeStorageFactory      LikeStorageFactory
	indexStorageFactory     IndexStorageFactory
	action                  chan Action
}

// Compile-time type check
var _ = (Namespace)((*likeNamespace)(nil))

// A DTO for creating a LikeNamespace.
type LikeNamespaceDto struct {
	Name                valueobjects.NamespaceName
	MaxSimilarProfiles  uint
	DislikeFactor       float32
	BasePath            string
	DeltaStorageFactory DeltaStorageFactory
	LikeStorageFactory  LikeStorageFactory
	IndexStorageFactory IndexStorageFactory
}

// Creates a new namespace.
func NewLikeNamespace(dto *LikeNamespaceDto) *likeNamespace {
	ns := &likeNamespace{
		name:                    dto.Name,
		maxSimilarProfiles:      dto.MaxSimilarProfiles,
		dislikeFactor:           dto.DislikeFactor,
		deltaStorageFactory:     dto.DeltaStorageFactory,
		likeStorageFactory:      dto.LikeStorageFactory,
		indexStorageFactory:     dto.IndexStorageFactory,
		basePath:                dto.BasePath,
		actionQueueFillWaitTime: time.Millisecond * 50,
		action:                  make(chan Action, 100),
	}
	// Set defaults
	if ns.maxSimilarProfiles == 0 {
		ns.maxSimilarProfiles = 1000
	}
	return ns
}

// Returns the name of the namespace.
func (ns *likeNamespace) GetName() valueobjects.NamespaceName {
	return ns.name
}

// Returns namespace subtype.
func (ns *likeNamespace) GetType() valueobjects.NamespaceType {
	return valueobjects.MakeLikeNamespaceType()
}

// Renames the namespace.
func (ns *likeNamespace) Rename(name valueobjects.NamespaceName) chan error {
	ns.name = name
	// TODO: rename the files
	return nil
}

// Changes maximum number of similar profiles to be used by recommendation algorithm.
func (ns *likeNamespace) SetMaxSimilarProfiles(limit uint) {
	ns.maxSimilarProfiles = limit
}

// Returns maximum number of similar profiles to be used by recommendation algorithm.
func (ns *likeNamespace) GetMaxSimilarProfiles() uint {
	return ns.maxSimilarProfiles
}

// Changes how much dislikes affect similarity of profiles.
// The value ranges from 0 to 1. 0.5 means that likes and dislikes has equal
// effect on similarity.
func (ns *likeNamespace) SetDislikeFactor(value float32) {
	ns.dislikeFactor = value
}

// Opens delta storage and recovers it if it is needed.
func (ns *likeNamespace) openMaybeRecoverDeltaStorage() (DeltaStorage, error) {
	filePath := ns.basePath + ns.name.Value() + ".delta"
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open delta file %s: %w", filePath, err)
	}
	storage, err := ns.deltaStorageFactory.OpenMaybeRecover(file)
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to open delta storage for %s: %w", ns.name.Value(), err)
	}
	return storage, nil
}

// Opens delta storage and recovers it if it is needed.
func (ns *likeNamespace) openMaybeResetIndexStorage() (IndexStorage, error) {
	filePath := ns.basePath + ns.name.Value() + ".index"
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open delta file %s: %w", filePath, err)
	}
	storage, err := ns.indexStorageFactory.Open(file, file)
	if err != nil {
		if !errors.Is(err, NewCorruptedFileError()) {
			file.Close()
			return nil, fmt.Errorf("failed to open delta storage for %s: %w", ns.name.Value(), err)
		}
		err = file.Truncate(0)
		if err != nil {
			file.Close()
			return nil, fmt.Errorf("failed to truncate %s: %w", filePath, err)
		}
		storage, err = ns.indexStorageFactory.Open(file, file)
		if err != nil {
			file.Close()
			return nil, fmt.Errorf("failed to open index file %s: %w", filePath, err)
		}
	}
	return storage, nil
}

// Opens delta storage and recovers it if it is needed.
func (ns *likeNamespace) openMaybeRecoverLikeStorage(
	deltaStorage DeltaStorage,
	indexStorage IndexStorage,
) (LikeStorage, error) {
	filePath := ns.basePath + ns.name.Value() + ".recdb"
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open recdb file %s: %w", filePath, err)
	}
	storage, err := ns.likeStorageFactory.OpenMaybeRecover(file, deltaStorage, indexStorage)
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to open like storage for %s: %w", ns.name.Value(), err)
	}
	return storage, nil
}

// Starts a separate thread to run the work on.
func (ns *likeNamespace) Start(ctx context.Context) error {
	deltaStorage, err := ns.openMaybeRecoverDeltaStorage()
	if err != nil {
		return err
	}
	indexStorage, err := ns.openMaybeResetIndexStorage()
	if err != nil {
		deltaStorage.Close()
		return err
	}
	likeStorage, err := ns.openMaybeRecoverLikeStorage(deltaStorage, indexStorage)
	if err != nil {
		deltaStorage.Close()
		indexStorage.Close()
		return err
	}
	go func() {
		defer likeStorage.Close()
		defer deltaStorage.Close()
		defer indexStorage.Close()
		defer log.Printf("LikeNamespace %s stopped\n", ns.name)
		for {
			select {
			case <-ctx.Done():
				ns.sendStoppedLikeNamespaceErrorToActionWaiters(nil)
				return
			case action, more := <-ns.action:
				if !more {
					ns.sendStoppedLikeNamespaceErrorToActionWaiters(nil)
					return
				}
				// We wan't to process as many actions as possible at a time.
				// But not too much, though.
				time.Sleep(ns.actionQueueFillWaitTime)
				// Take out actions from the channel
				actions := make([]Action, 1, len(ns.action)+1)
				actions[0] = action
				// Warning. We cannot iterate through the queue itself here!
				for i := 1; i < len(actions); i++ {
					actions[i], more = <-ns.action
					if !more || actions[i].ActionType == ActionStop {
						ns.sendStoppedLikeNamespaceErrorToActionWaiters(&actions)
						return
					}
				}
				likeStorage.ProcessActions(actions)
			}
		}
	}()
	return nil
}

// Sends a quit signal to the job worker thread started by a call to Run().
func (ns *likeNamespace) Stop() {
	log.Printf("Stopping namespace %s...\n", ns.name)
	close(ns.action)
}

func (ns *likeNamespace) sendStoppedLikeNamespaceErrorToActionWaiters(takenActions *[]Action) {
	// Get actions from the buffer and send an error to each one
	for {
		action, more := <-ns.action
		if !more {
			break
		}
		action.Error <- errors.New("the namespace stopped")
	}
	// Send an error to each action that has been taken out from the buffer
	// and should has been processed
	if takenActions != nil {
		for _, action := range *takenActions {
			action.Error <- errors.New("the namespace stopped")
		}
	}
}

// Removes the profile by its ID.
// If there is no profile with this ID found, it's NOT considered an error.
func (ns *likeNamespace) DeleteProfile(user uint64) error {
	err := make(chan error)
	ns.action <- Action{ActionDeleteProfile, err, DeleteProfilePayload{user}}
	return <-err
}

// Returns the profile by its ID or nil if it isn't found.
// If there is no profile with this ID found, it's NOT considered an error.
func (ns *likeNamespace) GetProfile(user uint64) (*Profile, error) {
	errChan := make(chan error)
	profileChan := make(chan *Profile)
	ns.action <- Action{
		ActionGetProfile,
		errChan,
		GetProfilePayload{user, profileChan},
	}
	select {
	case err := <-errChan:
		return nil, err
	case profile := <-profileChan:
		return profile, nil
	}
}

// Sets an item of the profile liked.
func (ns *likeNamespace) Like(user uint64, item uint64) error {
	err := make(chan error)
	ns.action <- Action{ActionLike, err, LikePayload{user, item}}
	return <-err
}

// Sets an item of the profile disliked.
func (ns *likeNamespace) Dislike(user uint64, item uint64) error {
	err := make(chan error)
	ns.action <- Action{ActionDislike, err, DislikePayload{user, item}}
	return <-err
}

// Sets an item of the profile undefined (not liked nor disliked).
func (ns *likeNamespace) DeleteItem(user uint64, item uint64) error {
	err := make(chan error)
	ns.action <- Action{ActionDeleteItem, err, DeleteItemPayload{user, item}}
	return <-err
}

// Returns the most similar profiles to the given one.
func (ns *likeNamespace) GetSimilarProfiles(user uint64) (*[]SimilarProfile, error) {
	errChan := make(chan error)
	profilesChan := make(chan *[]SimilarProfile)
	ns.action <- Action{
		ActionGetSimilarProfiles,
		errChan,
		GetSimilarProfilesPayload{user, profilesChan},
	}
	select {
	case err := <-errChan:
		return nil, err
	case profiles := <-profilesChan:
		return profiles, nil
	}
}

// Returns the recommended items for the user.
func (ns *likeNamespace) RecommendItems(user uint64) (*[]RecItem, error) {
	errChan := make(chan error)
	recsChan := make(chan *[]RecItem)
	ns.action <- Action{
		ActionRecommendItems,
		errChan,
		RecommendItemsPayload{user, recsChan},
	}
	select {
	case err := <-errChan:
		return nil, err
	case recs := <-recsChan:
		return recs, nil
	}
}
