package helpers

type unsignedinteger interface {
	uint8 | uint16 | uint32 | int64
}

type signedinteger interface {
	int | int8 | int16 | int32 | uint64
}

type integer interface {
	signedinteger | unsignedinteger
}

type float interface {
	float32 | float64
}

type Numeric interface {
	integer | float
}

type ordered interface {
	Numeric | ~string
}

// Returns index of the element in the array
func IndexOf[T comparable](haystack []T, needle T) int {
	for i, v := range haystack {
		if v == needle {
			return i
		}
	}
	return -1
}

// Returns index of the element in the sorted array
func BinaryIndexOf[T ordered](haystack []T, needle T) int {
	if len(haystack) == 0 {
		return -1
	}
	a, b, m := 0, len(haystack)-1, 0
	if haystack[b] == needle {
		return b
	}
	if haystack[a] == needle {
		return a
	}
	for a < b {
		m = (a + b) / 2
		if needle == haystack[m] {
			return m
		}
		if needle < haystack[m] {
			if b == m {
				break
			}
			b = m
		} else {
			if a == m {
				break
			}
			a = m
		}
	}
	return -1
}

// Returns abs(a - b) that is valid for unsigned integers too.
func Deviation[T Numeric](a, b T) T {
	if a < b {
		return b - a
	}
	return a - b
}

// Returns the index of the element whose value in the sorted array is the
// closest to the given one.
func BinaryIndexOfClosest[T Numeric](haystack []T, needle T) int {
	if len(haystack) == 0 {
		return -1
	}
	a, b, m := 0, len(haystack)-1, 0
	minDIndex, minD := a, Deviation(haystack[a], needle)
	bD := Deviation(haystack[b], needle)
	if bD < minD {
		minD = bD
		minDIndex = b
	}
	if minD == 0 {
		return minDIndex
	}
	for a < b {
		m = (a + b) / 2
		if needle == haystack[m] {
			return m
		}
		if d := Deviation(haystack[m], needle); d < minD {
			minD = d
			minDIndex = m
		}
		if needle < haystack[m] {
			if b == m {
				break
			}
			b = m
		} else {
			if a == m {
				break
			}
			a = m
		}
	}
	return minDIndex
}

// Removes the element from the array by element index.
// The function doesn't preserves order of the elements.
func Remove[T any](haystack []T, index int) []T {
	haystack[index] = haystack[len(haystack)-1]
	return haystack[:len(haystack)-1]
}

// Removes the element from the array by element index.
// The function preserves order of the elements.
func RemoveSavingOrder[T comparable](haystack []T, index int) []T {
	return append(haystack[:index], haystack[index+1:]...)
}

// Inserts the element to the array.
func Insert[T Numeric](haystack []T, index int, value T) []T {
	if len(haystack) == index { // nil or empty slice or after last element
		return append(haystack, value)
	}
	haystack = append(haystack[:index+1], haystack[index:]...) // index < len(a)
	haystack[index] = value
	return haystack
}

// Inserts the element to the array keeping it ordered.
// The function preserves order of the elements.
func InsertSavingOrder[T Numeric](haystack []T, value T) []T {
	if len(haystack) == 0 {
		return append(haystack, value)
	}
	index := BinaryIndexOfClosest(haystack, value)
	if value >= haystack[index] {
		return Insert(haystack, index+1, value)
	}
	return Insert(haystack, index, value)
}

// Inserts the element to the array keeping it ordered if there is no element
// of this value in the array yet.
// The function preserves order of the elements.
func PutSavingOrder[T Numeric](haystack []T, value T) []T {
	if len(haystack) == 0 {
		return append(haystack, value)
	}
	index := BinaryIndexOfClosest(haystack, value)
	if value == haystack[index] {
		return haystack
	}
	if value > haystack[index] {
		return Insert(haystack, index+1, value)
	}
	return Insert(haystack, index, value)
}
