package like

import (
	"fmt"
	"recengine/internal/domain/entities"
)

type Storage struct {
	domainName string
}

// Compile-type type check
var _ = (entities.LikeStorage)((*Storage)(nil))

func (s *Storage) GetDeltaFileSize() uint64 {
	return 0
}

func (s *Storage) ApplyDelta() error {
	return nil
}

func (s *Storage) ProcessActions(actions []entities.Action) error {
	for _, action := range actions {
		switch action.ActionType {
		case entities.ActionDeleteProfile:
			action.Error <- nil
			// TODO
		case entities.ActionGetProfile:
			payload := action.Payload.(entities.GetProfilePayload)
			payload.Profile <- nil
			// TODO
		case entities.ActionLike:
			action.Error <- nil
			// TODO
		case entities.ActionDislike:
			action.Error <- nil
			// TODO
		case entities.ActionDeleteItem:
			action.Error <- nil
			// TODO
		case entities.ActionGetSimilarProfiles:
			payload := action.Payload.(entities.GetSimilarProfilesPayload)
			payload.Profiles <- &[]entities.SimilarProfile{}
			// TODO
		case entities.ActionRecommendItems:
			payload := action.Payload.(entities.RecommendItemsPayload)
			payload.Items <- &[]entities.RecItem{}
			// TODO
		default:
			action.Error <- fmt.Errorf(
				"unknown action %d for DB %s",
				action.ActionType,
				s.domainName,
			)
		}
	}
	return nil
}
