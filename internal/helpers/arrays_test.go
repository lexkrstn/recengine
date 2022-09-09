package helpers

import (
	"reflect"
	"testing"
)

func TestIndexOf(t *testing.T) {
	type Fixture struct {
		haystack []int
		needle   int
		expected int
	}
	fixtures := []Fixture{
		// Odd-length array
		{[]int{1, 10, 13, 15, 23}, 1, 0},
		{[]int{1, 10, 13, 15, 23}, 23, 4},
		{[]int{1, 10, 13, 15, 23}, -42, -1},
		{[]int{1, 10, 13, 15, 23}, 42, -1},
		{[]int{1, 10, 13, 15, 23}, 13, 2},
		{[]int{1, 10, 13, 15, 23}, 15, 3},
		// Even-length array
		{[]int{1, 10, 13, 15}, 1, 0},
		{[]int{1, 10, 13, 15}, 15, 3},
		{[]int{1, 10, 13, 15}, -42, -1},
		{[]int{1, 10, 13, 15}, 42, -1},
		// Other arrays
		{[]int{}, 42, -1},
		{[]int{1}, 1, 0},
		{[]int{1}, 42, -1},
		{[]int{1, 10}, 10, 1},
		{[]int{1, 10}, 42, -1},
	}
	for _, fixture := range fixtures {
		got := IndexOf(fixture.haystack, fixture.needle)
		if got != fixture.expected {
			t.Errorf(
				"IndexOf(%v, %d) = %d; want %d",
				fixture.haystack,
				fixture.needle,
				got,
				fixture.expected,
			)
		}
	}
}

func TestBinaryIndexOf(t *testing.T) {
	type Fixture struct {
		haystack []int
		needle   int
		expected int
	}
	fixtures := []Fixture{
		// Odd-length array
		{[]int{1, 10, 13, 15, 23}, 1, 0},
		{[]int{1, 10, 13, 15, 23}, 23, 4},
		{[]int{1, 10, 13, 15, 23}, -42, -1},
		{[]int{1, 10, 13, 15, 23}, 42, -1},
		{[]int{1, 10, 13, 15, 23}, 13, 2},
		{[]int{1, 10, 13, 15, 23}, 15, 3},
		// Even-length array
		{[]int{1, 10, 13, 15}, 1, 0},
		{[]int{1, 10, 13, 15}, 15, 3},
		{[]int{1, 10, 13, 15}, -42, -1},
		{[]int{1, 10, 13, 15}, 42, -1},
		// Other arrays
		{[]int{}, 42, -1},
		{[]int{1}, 1, 0},
		{[]int{1}, 42, -1},
		{[]int{1, 10}, 10, 1},
		{[]int{1, 10}, 42, -1},
	}
	for _, fixture := range fixtures {
		got := BinaryIndexOf(fixture.haystack, fixture.needle)
		if got != fixture.expected {
			t.Errorf(
				"BinaryIndexOf(%v, %d) = %d; want %d",
				fixture.haystack,
				fixture.needle,
				got,
				fixture.expected,
			)
		}
	}
}

func TestBinaryIndexOfClosest(t *testing.T) {
	type Fixture struct {
		haystack []int
		needle   int
		expected int
	}
	fixtures := []Fixture{
		{[]int{1, 10, 13, 15, 23}, 1, 0},
		{[]int{1, 10, 13, 15, 23}, 23, 4},
		{[]int{1, 10, 13, 15, 23}, -42, 0},
		{[]int{1, 10, 13, 15, 23}, 42, 4},
		{[]int{1, 10, 13, 15, 23}, 13, 2},
		{[]int{1, 10, 13, 15, 23}, 14, 2},
		{[]int{1, 10, 13, 15, 23}, 16, 3},
		{[]int{1, 10, 13, 15, 23}, 20, 4},
		{[]int{1, 10, 13, 15, 23, 53}, 25, 4},
		{[]int{1, 10, 13, 15, 23, 53}, 41, 5},
	}
	for _, fixture := range fixtures {
		got := BinaryIndexOfClosest(fixture.haystack, fixture.needle)
		if got != fixture.expected {
			t.Errorf(
				"BinaryIndexOfClosest(%v, %d) = %d; want %d",
				fixture.haystack,
				fixture.needle,
				got,
				fixture.expected,
			)
		}
	}
}

func TestRemoveSavingOrder(t *testing.T) {
	type Fixture struct {
		haystack []int
		index    int
		expected []int
	}
	fixtures := []Fixture{
		{[]int{1, 10, 13, 15, 23}, 0, []int{10, 13, 15, 23}},
		{[]int{1, 10, 13, 15, 23}, 4, []int{1, 10, 13, 15}},
		{[]int{1, 10, 13, 15, 23}, 2, []int{1, 10, 15, 23}},
	}
	for _, fixture := range fixtures {
		got := RemoveSavingOrder(fixture.haystack, fixture.index)
		if !reflect.DeepEqual(got, fixture.expected) {
			t.Errorf(
				"RemoveSavingOrder(%v, %d) = %d; want %d",
				fixture.haystack,
				fixture.index,
				got,
				fixture.expected,
			)
		}
	}
}

func TestInsertSavingOrder(t *testing.T) {
	type Fixture struct {
		haystack []int
		value    int
		expected []int
	}
	fixtures := []Fixture{
		{[]int{1, 10, 13, 15}, -42, []int{-42, 1, 10, 13, 15}},
		{[]int{1, 10, 13, 15}, 42, []int{1, 10, 13, 15, 42}},
		{[]int{1, 10, 13, 15}, 1, []int{1, 1, 10, 13, 15}},
		{[]int{1, 10, 13, 15}, 15, []int{1, 10, 13, 15, 15}},
		{[]int{1, 10, 13, 15}, 3, []int{1, 3, 10, 13, 15}},
		{[]int{1, 10, 13, 15}, 14, []int{1, 10, 13, 14, 15}},
		{[]int{1, 10, 13, 15}, 12, []int{1, 10, 12, 13, 15}},
		{[]int{1, 10, 13, 15}, 13, []int{1, 10, 13, 13, 15}},
		{[]int{1, 10}, 5, []int{1, 5, 10}},
		{[]int{1}, -5, []int{-5, 1}},
		{[]int{1}, 5, []int{1, 5}},
		{[]int{1}, 1, []int{1, 1}},
		{[]int{}, 5, []int{5}},
	}
	for _, fixture := range fixtures {
		got := InsertSavingOrder(fixture.haystack, fixture.value)
		if !reflect.DeepEqual(got, fixture.expected) {
			t.Errorf(
				"InsertSavingOrder(%v, %d) = %d; want %d",
				fixture.haystack,
				fixture.value,
				got,
				fixture.expected,
			)
		}
	}
}

func TestPutSavingOrder(t *testing.T) {
	type Fixture struct {
		haystack []int
		value    int
		expected []int
	}
	fixtures := []Fixture{
		{[]int{1, 10, 13, 15}, -42, []int{-42, 1, 10, 13, 15}},
		{[]int{1, 10, 13, 15}, 42, []int{1, 10, 13, 15, 42}},
		{[]int{1, 10, 13, 15}, 1, []int{1, 10, 13, 15}},
		{[]int{1, 10, 13, 15}, 15, []int{1, 10, 13, 15}},
		{[]int{1, 10, 13, 15}, 3, []int{1, 3, 10, 13, 15}},
		{[]int{1, 10, 13, 15}, 14, []int{1, 10, 13, 14, 15}},
		{[]int{1, 10, 13, 15}, 12, []int{1, 10, 12, 13, 15}},
		{[]int{1, 10, 13, 15}, 13, []int{1, 10, 13, 15}},
		{[]int{1, 10}, 5, []int{1, 5, 10}},
		{[]int{1}, -5, []int{-5, 1}},
		{[]int{1}, 5, []int{1, 5}},
		{[]int{1}, 1, []int{1}},
		{[]int{}, 5, []int{5}},
	}
	for _, fixture := range fixtures {
		got := PutSavingOrder(fixture.haystack, fixture.value)
		if !reflect.DeepEqual(got, fixture.expected) {
			t.Errorf(
				"PutSavingOrder(%v, %d) = %d; want %d",
				fixture.haystack,
				fixture.value,
				got,
				fixture.expected,
			)
		}
	}
}
