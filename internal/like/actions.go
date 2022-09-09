package like

type actionType = int

const (
	actionStop               actionType = iota
	actionDeleteProfile      actionType = iota
	actionGetProfile         actionType = iota
	actionLike               actionType = iota
	actionDislike            actionType = iota
	actionDeleteItem         actionType = iota
	actionGetSimilarProfiles actionType = iota
	actionRecommendItems     actionType = iota
)

type Action struct {
	actionType actionType
	error      chan error
	payload    any
}

type deleteProfilePayload struct {
	user uint64
}

type getProfilePayload struct {
	user    uint64
	profile chan *Profile
}

type likePayload struct {
	user uint64
	item uint64
}

type dislikePayload = likePayload
type deleteItemPayload = likePayload

type SimilarProfile struct {
	Profile    *Profile
	Similarity float32
}

type getSimilarProfilesPayload struct {
	user     uint64
	profiles chan *[]SimilarProfile
}

type ItemRecommendation struct {
	Item      uint64
	Relevance float32
}

type recommendItemsPayload struct {
	user            uint64
	recommendations chan *[]ItemRecommendation
}
