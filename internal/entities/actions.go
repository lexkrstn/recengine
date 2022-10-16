package entities

type ActionType = int

const (
	ActionStop               ActionType = iota
	ActionDeleteProfile      ActionType = iota
	ActionGetProfile         ActionType = iota
	ActionLike               ActionType = iota
	ActionDislike            ActionType = iota
	ActionDeleteItem         ActionType = iota
	ActionGetSimilarProfiles ActionType = iota
	ActionRecommendItems     ActionType = iota
)

type Action struct {
	ActionType ActionType
	Error      chan error
	Payload    any
}

type DeleteProfilePayload struct {
	UserID uint64
}

type GetProfilePayload struct {
	UserID  uint64
	Profile chan *Profile
}

type LikePayload struct {
	UserID uint64
	ItemID uint64
}

type DislikePayload = LikePayload
type DeleteItemPayload = LikePayload

type GetSimilarProfilesPayload struct {
	UserID   uint64
	Profiles chan *[]SimilarProfile
}

type RecommendItemsPayload struct {
	UserID uint64
	Items  chan *[]RecItem
}
