package like

import (
	"math"
	"testing"
)

func TestProfileLikingUnliking(t *testing.T) {
	p := Profile{}
	if got := p.QualifyItem(11); got != ItemUnknown {
		t.Errorf("QualifyItem(11) = %d; want %d", got, ItemUnknown)
	}
	p.Like(11)
	if got := p.QualifyItem(11); got != ItemLiked {
		t.Errorf("QualifyItem(11) = %d; want %d", got, ItemLiked)
	}
	p.Like(11)
	p.Like(12)
	p.Unlike(11)
	if got := p.QualifyItem(11); got != ItemUnknown {
		t.Errorf("QualifyItem(11) = %d; want %d", got, ItemUnknown)
	}
	if got := p.QualifyItem(12); got != ItemLiked {
		t.Errorf("QualifyItem(12) = %d; want %d", got, ItemLiked)
	}
	p.Undislike(12)
	if got := p.QualifyItem(12); got != ItemLiked {
		t.Errorf("QualifyItem(12) = %d; want %d", got, ItemLiked)
	}
}

func TestProfileDislikingUndisliking(t *testing.T) {
	p := Profile{}
	if got := p.QualifyItem(11); got != ItemUnknown {
		t.Errorf("QualifyItem(11) = %d; want %d", got, ItemUnknown)
	}
	p.Dislike(11)
	if got := p.QualifyItem(11); got != ItemDisliked {
		t.Errorf("QualifyItem(11) = %d; want %d", got, ItemDisliked)
	}
	p.Dislike(11)
	p.Dislike(12)
	p.Undislike(11)
	if got := p.QualifyItem(11); got != ItemUnknown {
		t.Errorf("QualifyItem(11) = %d; want %d", got, ItemUnknown)
	}
	if got := p.QualifyItem(12); got != ItemDisliked {
		t.Errorf("QualifyItem(12) = %d; want %d", got, ItemDisliked)
	}
	p.Unlike(12)
	if got := p.QualifyItem(12); got != ItemDisliked {
		t.Errorf("QualifyItem(12) = %d; want %d", got, ItemDisliked)
	}
}

func TestProfileComputeSimilarty(t *testing.T) {
	type Fixture struct {
		likesA        []uint64
		likesB        []uint64
		dislikesA     []uint64
		dislikesB     []uint64
		dislikeFactor float32
		expected      float32
	}
	fixtures := []Fixture{
		{
			[]uint64{1, 2, 3}, []uint64{1, 2, 3},
			[]uint64{10, 20, 30}, []uint64{10, 20, 30},
			1, 100,
		},
		{
			[]uint64{1, 2, 3}, []uint64{4, 5, 6},
			[]uint64{10, 20, 30}, []uint64{40, 50, 60},
			1, 0,
		},
		{
			[]uint64{1, 2, 3}, []uint64{1, 2, 4},
			[]uint64{10, 20, 30}, []uint64{10, 20, 40},
			1, 50,
		},
		{
			[]uint64{1, 2, 3}, []uint64{1, 3, 4},
			[]uint64{10, 20, 30}, []uint64{10, 30, 40},
			1, 50,
		},
		{
			[]uint64{1, 2, 3}, []uint64{1, 3, 4, 5},
			[]uint64{10, 20, 30}, []uint64{10, 30, 40, 50},
			1, 40,
		},
	}
	for _, fixture := range fixtures {
		a := Profile{1, fixture.likesA, fixture.dislikesA}
		b := Profile{2, fixture.likesB, fixture.dislikesB}
		got := a.ComputeSimilarity(b, 1)
		if math.Abs(float64(got-fixture.expected)) > float64(0.001) {
			t.Errorf(
				"{{%v}{%v}}.ComputeSimilarity({{%v}{%v}},%f) = %f; want %f",
				fixture.likesA,
				fixture.dislikesA,
				fixture.likesB,
				fixture.dislikesB,
				fixture.dislikeFactor,
				got,
				fixture.expected,
			)
		}
	}
}
