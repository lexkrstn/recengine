package entities

import (
	"context"
	"errors"
	"log"
	"recengine/internal/valueobjects"
	"time"
)

// LikeNamespaces perform the same function as databases in relational databases.
type LikeNamespace struct {
	Name                    valueobjects.NamespaceName
	MaxSimilarProfiles      uint
	DislikeFactor           float32
	storage                 LikeStorage
	actionQueueFillWaitTime time.Duration
	action                  chan Action
}

// A DTO for creating a LikeNamespace.
type LikeNamespaceDto struct {
	Name               valueobjects.NamespaceName
	MaxSimilarProfiles uint
	DislikeFactor      float32
}

// Creates a new namespace.
func NewLikeNamespace(dto *LikeNamespaceDto) *LikeNamespace {
	ns := &LikeNamespace{
		Name:                    dto.Name,
		MaxSimilarProfiles:      dto.MaxSimilarProfiles,
		DislikeFactor:           dto.DislikeFactor,
		actionQueueFillWaitTime: time.Millisecond * 50,
		action:                  make(chan Action, 100),
	}
	// Set defaults
	if ns.MaxSimilarProfiles == 0 {
		ns.MaxSimilarProfiles = 1000
	}
	return ns
}

// Returns the name of the namespace.
func (ns *LikeNamespace) GetName() valueobjects.NamespaceName {
	return ns.Name
}

// Returns namespace subtype.
func (ns *LikeNamespace) GetType() valueobjects.NamespaceType {
	return valueobjects.MakeLikeNamespaceType()
}

// Renames the namespace.
func (ns *LikeNamespace) Rename(name valueobjects.NamespaceName) chan error {
	ns.Name = name
	// TODO: rename the files
	return nil
}

// Changes maximum number of similar profiles to be used by recommendation algorithm.
func (ns *LikeNamespace) SetMaxSimilarProfiles(limit uint) {
	ns.MaxSimilarProfiles = limit
}

// Returns maximum number of similar profiles to be used by recommendation algorithm.
func (ns *LikeNamespace) GetMaxSimilarProfiles() uint {
	return ns.MaxSimilarProfiles
}

// Changes how much dislikes affect similarity of profiles.
// The value ranges from 0 to 1. 0.5 means that likes and dislikes has equal
// effect on similarity.
func (ns *LikeNamespace) SetDislikeFactor(value float32) {
	ns.DislikeFactor = value
}

// Starts a separate thread to run the work on.
func (ns *LikeNamespace) Start(ctx context.Context) {
	go func() {
		defer log.Printf("LikeNamespace %s stopped\n", ns.Name)
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
				ns.storage.ProcessActions(actions)
			}
		}
	}()
}

// Sends a quit signal to the job worker thread started by a call to Run().
func (ns *LikeNamespace) Stop() {
	log.Printf("Stopping namespace %s...\n", ns.Name)
	close(ns.action)
}

func (ns *LikeNamespace) sendStoppedLikeNamespaceErrorToActionWaiters(takenActions *[]Action) {
	// Get actions from the buffer and send an error to each one
	for {
		action, more := <-ns.action
		if !more {
			break
		}
		action.Error <- errors.New("The namespace stopped")
	}
	// Send an error to each action that has been taken out from the buffer
	// and should has been processed
	if takenActions != nil {
		for _, action := range *takenActions {
			action.Error <- errors.New("The namespace stopped")
		}
	}
}

// Removes the profile by its ID.
// If there is no profile with this ID found, it's NOT considered an error.
func (ns *LikeNamespace) DeleteProfile(user uint64) error {
	err := make(chan error)
	ns.action <- Action{ActionDeleteProfile, err, DeleteProfilePayload{user}}
	return <-err
}

// Returns the profile by its ID or nil if it isn't found.
// If there is no profile with this ID found, it's NOT considered an error.
func (ns *LikeNamespace) GetProfile(user uint64) (*Profile, error) {
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
func (ns *LikeNamespace) Like(user uint64, item uint64) error {
	err := make(chan error)
	ns.action <- Action{ActionLike, err, LikePayload{user, item}}
	return <-err
}

// Sets an item of the profile disliked.
func (ns *LikeNamespace) Dislike(user uint64, item uint64) error {
	err := make(chan error)
	ns.action <- Action{ActionDislike, err, DislikePayload{user, item}}
	return <-err
}

// Sets an item of the profile undefined (not liked nor disliked).
func (ns *LikeNamespace) DeleteItem(user uint64, item uint64) error {
	err := make(chan error)
	ns.action <- Action{ActionDeleteItem, err, DeleteItemPayload{user, item}}
	return <-err
}

// Returns the most similar profiles to the given one.
func (ns *LikeNamespace) GetSimilarProfiles(user uint64) (*[]SimilarProfile, error) {
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
func (ns *LikeNamespace) RecommendItems(user uint64) (*[]RecItem, error) {
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
