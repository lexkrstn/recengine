package recdb

import (
	"fmt"
	"recengine/internal/domain"
)

type likeStorage struct {
}

// Compile-time type check
var _ = (domain.LikeStorage)((*likeStorage)(nil))

// Returns new like storage instance.
func NewLikeStorage() domain.LikeStorage {
	return &likeStorage{}
}

func (s *likeStorage) Close() error {
	// TODO
	return nil
}

func (s *likeStorage) ProcessActions(actions []domain.Action) error {
	for _, action := range actions {
		switch action.ActionType {
		case domain.ActionDeleteProfile:
			action.Error <- nil
			// TODO
		case domain.ActionGetProfile:
			payload := action.Payload.(domain.GetProfilePayload)
			payload.Profile <- nil
			// TODO
		case domain.ActionLike:
			action.Error <- nil
			// TODO
		case domain.ActionDislike:
			action.Error <- nil
			// TODO
		case domain.ActionDeleteItem:
			action.Error <- nil
			// TODO
		case domain.ActionGetSimilarProfiles:
			payload := action.Payload.(domain.GetSimilarProfilesPayload)
			payload.Profiles <- &[]domain.SimilarProfile{}
			// TODO
		case domain.ActionRecommendItems:
			payload := action.Payload.(domain.RecommendItemsPayload)
			payload.Items <- &[]domain.RecItem{}
			// TODO
		default:
			action.Error <- fmt.Errorf("unknown action %d", action.ActionType)
		}
	}
	return nil
}
