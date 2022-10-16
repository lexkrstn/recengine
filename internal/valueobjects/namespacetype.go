package valueobjects

import "fmt"

const (
	NamespaceTypeLike string = "like"
)

type NamespaceType struct {
	value string
}

func ParseNamespaceType(value string) (NamespaceType, error) {
	t := NamespaceType{value}
	if value != NamespaceTypeLike {
		return t, fmt.Errorf("Invalid namespace type '%s'", value)
	}
	return t, nil
}

func MakeLikeNamespaceType() NamespaceType {
	return NamespaceType{NamespaceTypeLike}
}

func (t NamespaceType) Value() string {
	return t.value
}
