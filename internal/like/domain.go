package like

import (
	"context"
	"errors"
	"log"
	"time"
)

// Domains perform the same function as databases in relational databases.
type Domain struct {
	Name                    string  `json:"name"`
	Type                    string  `json:"type"`
	MaxSimilarProfiles      int     `json:"maxSimilarProfiles"`
	DislikeFactor           float32 `json:"dislikeFactor"`
	database                *database
	actionQueueFillWaitTime time.Duration
	action                  chan Action
}

// A DTO for creating a Domain.
type DomainCreateInput struct {
	Name               string
	Type               string
	MaxSimilarProfiles int
	DislikeFactor      float32
}

// A DTO for updating a Domain.
type DomainUpdateInput = struct {
	Name               string
	MaxSimilarProfiles int
	DislikeFactor      float32
}

// Creates a new domain.
func NewDomain(dto *DomainCreateInput) *Domain {
	domain := &Domain{
		Name:                    dto.Name,
		Type:                    dto.Type,
		MaxSimilarProfiles:      dto.MaxSimilarProfiles,
		DislikeFactor:           dto.DislikeFactor,
		actionQueueFillWaitTime: time.Millisecond * 50,
		action:                  make(chan Action, 100),
	}
	// Set defaults
	if domain.MaxSimilarProfiles == 0 {
		domain.MaxSimilarProfiles = 1000
	}
	return domain
}

// Returns the name of the domain.
func (domain *Domain) GetName() string {
	return domain.Name
}

// Renames the domain.
func (domain *Domain) Rename(name string) chan error {
	domain.Name = name
	// TODO: rename the files
	return nil
}

// Changes maximum number of similar profiles to be used by recommendation algorithm.
func (domain *Domain) SetMaxSimilarProfiles(value int) {
	domain.MaxSimilarProfiles = value
}

// Changes how much dislikes affect similarity of profiles.
// The value ranges from 0 to 1. 0.5 means that likes and dislikes has equal
// effect on similarity.
func (domain *Domain) SetDislikeFactor(value float32) {
	domain.DislikeFactor = value
}

// Starts a separate thread to run the work on.
func (domain *Domain) Start(ctx context.Context) {
	go func() {
		defer log.Printf("Domain %s stopped\n", domain.Name)
		for {
			select {
			case <-ctx.Done():
				domain.sendStoppedDomainErrorToActionWaiters(nil)
				return
			case action, more := <-domain.action:
				if !more {
					domain.sendStoppedDomainErrorToActionWaiters(nil)
					return
				}
				// We wan't to process as many actions as possible at a time.
				// But not too much, though.
				time.Sleep(domain.actionQueueFillWaitTime)
				// Take out actions from the channel
				actions := make([]Action, 1, len(domain.action)+1)
				actions[0] = action
				// Warning. We cannot iterate through the queue itself here!
				for i := 1; i < len(actions); i++ {
					actions[i], more = <-domain.action
					if !more || actions[i].actionType == actionStop {
						domain.sendStoppedDomainErrorToActionWaiters(&actions)
						return
					}
				}
				domain.database.processActions(actions)
			}
		}
	}()
}

// Sends a quit signal to the job worker thread started by a call to Run().
func (domain *Domain) Stop() {
	log.Printf("Stopping domain %s...\n", domain.Name)
	close(domain.action)
}

func (domain *Domain) sendStoppedDomainErrorToActionWaiters(takenActions *[]Action) {
	// Get actions from the buffer and send an error to each one
	for {
		action, more := <-domain.action
		if !more {
			break
		}
		action.error <- errors.New("The domain stopped")
	}
	// Send an error to each action that has been taken out from the buffer
	// and should has been processed
	if takenActions != nil {
		for _, action := range *takenActions {
			action.error <- errors.New("The domain stopped")
		}
	}
}

// Removes the profile by its ID.
// If there is no profile with this ID found, it's NOT considered an error.
func (domain *Domain) DeleteProfile(user uint64) error {
	err := make(chan error)
	domain.action <- Action{actionDeleteProfile, err, deleteProfilePayload{user}}
	return <-err
}

// Returns the profile by its ID or nil if it isn't found.
// If there is no profile with this ID found, it's NOT considered an error.
func (domain *Domain) GetProfile(user uint64) (*Profile, error) {
	errChan := make(chan error)
	profileChan := make(chan *Profile)
	domain.action <- Action{
		actionGetProfile,
		errChan,
		getProfilePayload{user, profileChan},
	}
	select {
	case err := <-errChan:
		return nil, err
	case profile := <-profileChan:
		return profile, nil
	}
}

// Sets an item of the profile liked.
func (domain *Domain) Like(user uint64, item uint64) error {
	err := make(chan error)
	domain.action <- Action{actionLike, err, likePayload{user, item}}
	return <-err
}

// Sets an item of the profile disliked.
func (domain *Domain) Dislike(user uint64, item uint64) error {
	err := make(chan error)
	domain.action <- Action{actionDislike, err, dislikePayload{user, item}}
	return <-err
}

// Sets an item of the profile undefined (not liked nor disliked).
func (domain *Domain) DeleteItem(user uint64, item uint64) error {
	err := make(chan error)
	domain.action <- Action{actionDeleteItem, err, deleteItemPayload{user, item}}
	return <-err
}

// Returns the most similar profiles to the given one.
func (domain *Domain) GetSimilarProfiles(user uint64) (*[]SimilarProfile, error) {
	errChan := make(chan error)
	profilesChan := make(chan *[]SimilarProfile)
	domain.action <- Action{
		actionGetSimilarProfiles,
		errChan,
		getSimilarProfilesPayload{user, profilesChan},
	}
	select {
	case err := <-errChan:
		return nil, err
	case profiles := <-profilesChan:
		return profiles, nil
	}
}

// Returns the recommended items for the user.
func (domain *Domain) RecommendItems(user uint64) (*[]ItemRecommendation, error) {
	errChan := make(chan error)
	recsChan := make(chan *[]ItemRecommendation)
	domain.action <- Action{
		actionRecommendItems,
		errChan,
		recommendItemsPayload{user, recsChan},
	}
	select {
	case err := <-errChan:
		return nil, err
	case recs := <-recsChan:
		return recs, nil
	}
}
