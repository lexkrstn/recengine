package like

import (
	"fmt"
	"recengine/internal/entities"
)

type Storage struct {
	entities.LikeStorage
	domainName string
}

func (s *Storage) getDatabaseFilePath() string {
	return s.domainName + ".redb"
}

func (s *Storage) ProcessActions(actions []entities.Action) {
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
				"Unknown action %d for DB %s",
				action.ActionType,
				s.domainName,
			)
		}
	}
}
