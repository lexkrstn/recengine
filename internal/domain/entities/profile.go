package entities

import "recengine/internal/helpers"

const (
	ItemDisliked = -1
	ItemUnknown  = 0
	ItemLiked    = 1
)

// Encapsulates all the items liked by a user.
type Profile struct {
	// User or compilation ID.
	UserID uint64 `json:"user"`

	// Array of the IDs of the liked items.
	Likes []uint64 `json:"likes"`

	// Array of the IDs of the liked items.
	Dislikes []uint64 `json:"dislikes"`
}

// Creates new empty like profile object.
func NewProfile(UserID uint64) *Profile {
	return &Profile{
		UserID:   UserID,
		Likes:    make([]uint64, 0),
		Dislikes: make([]uint64, 0),
	}
}

// Returns 1 if the profile contains the item and it's liked, -1 if disliked and
// 0 if the profile doesn't have the item.
func (p *Profile) QualifyItem(item uint64) int {
	if helpers.BinaryIndexOf(p.Likes, item) >= 0 {
		return ItemLiked
	}
	if helpers.BinaryIndexOf(p.Dislikes, item) >= 0 {
		return ItemDisliked
	}
	return ItemUnknown
}

// Adds the item to the liked list of the profile.
func (p *Profile) Like(item uint64) {
	p.Likes = helpers.PutSavingOrder(p.Likes, item)
	p.Undislike(item)
}

// Removes the item from the liked list of the profile.
func (p *Profile) Unlike(item uint64) {
	index := helpers.BinaryIndexOf(p.Likes, item)
	if index >= 0 {
		p.Likes = helpers.RemoveSavingOrder(p.Likes, index)
	}
}

// Adds the item to the disliked list of the profile.
func (p *Profile) Dislike(item uint64) {
	p.Dislikes = helpers.PutSavingOrder(p.Dislikes, item)
	p.Unlike(item)
}

// Removes the item from the disliked list of the profile.
func (p *Profile) Undislike(item uint64) {
	index := helpers.BinaryIndexOf(p.Dislikes, item)
	if index >= 0 {
		p.Dislikes = helpers.RemoveSavingOrder(p.Dislikes, index)
	}
}

// Removes the item from the profile.
func (p *Profile) RemoveItem(item uint64) {
	p.Unlike(item)
	p.Undislike(item)
}

// Computes the degree of similarity between two sets as value in range [0..1].
// Given sets A and B, similarity rate = | A ^ B | / | A v B |
// The second returned value is the size of the union of the sets.
func computeSimilarityBetweenSets(itemsA []uint64, itemsB []uint64) (float32, int) {
	conjunction := 0
	for _, item := range itemsA {
		if helpers.BinaryIndexOf(itemsB, item) >= 0 {
			conjunction++
		}
	}
	disjunction := len(itemsA) + len(itemsB) - conjunction
	if disjunction == 0 {
		return 0, 0
	}
	return float32(conjunction) / float32(disjunction), disjunction
}

// Computes similarity between two profiles as value in range [0..100].
// The value is computed as weighted sum of similarities of likes and dislikes.
// The dislikeFactor is the value from 0 to 1 that may be used to change
// the contribution of dislikes to the result.
func (p1 *Profile) ComputeSimilarity(p2 Profile, dislikeFactor float32) float32 {
	likesSim, likesWeight := computeSimilarityBetweenSets(p1.Likes, p2.Likes)
	dislikesSim, dislikesWeight := computeSimilarityBetweenSets(p1.Dislikes, p2.Dislikes)
	dislikesWeight = int(float32(dislikesWeight) * dislikeFactor)
	return 100 * ((likesSim*float32(likesWeight) + dislikesSim*float32(dislikesWeight)) /
		(float32(likesWeight) + float32(dislikesWeight)))
}
