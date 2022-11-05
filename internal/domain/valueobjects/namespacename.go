package valueobjects

import (
	"errors"
	"regexp"
)

type NamespaceName struct {
	value string
}

func ParseNamespaceName(value string) (NamespaceName, error) {
	t := NamespaceName{value}
	if !regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_\-]*$`).MatchString(value) {
		return t, errors.New("invalid namespace name")
	}
	return t, nil
}

func (t NamespaceName) Value() string {
	return t.value
}
