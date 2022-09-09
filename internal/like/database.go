package like

import (
	"fmt"
)

type database struct {
	domainName string
}

func getDatabaseFilePathByDomainName(domainName string) string {
	return domainName + ".redb"
}

func (db *database) processActions(actions []Action) {
	for _, action := range actions {
		switch action.actionType {
		case actionDeleteProfile:
			action.error <- nil
			// TODO
		case actionGetProfile:
			payload := action.payload.(getProfilePayload)
			payload.profile <- nil
			// TODO
		case actionLike:
			action.error <- nil
			// TODO
		case actionDislike:
			action.error <- nil
			// TODO
		case actionDeleteItem:
			action.error <- nil
			// TODO
		case actionGetSimilarProfiles:
			payload := action.payload.(getSimilarProfilesPayload)
			payload.profiles <- &[]SimilarProfile{}
			// TODO
		case actionRecommendItems:
			payload := action.payload.(recommendItemsPayload)
			payload.recommendations <- &[]ItemRecommendation{}
			// TODO
		default:
			action.error <- fmt.Errorf(
				"Unknown action %d for DB %s",
				action.actionType,
				db.domainName,
			)
		}
	}
}

// func (db *database) readerIterator() (ReaderIterator[T io.Reader], error) {
// }
