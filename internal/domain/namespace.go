package domain

import (
	"context"
	"recengine/internal/domain/valueobjects"
)

// Interface that domains of any type must implement.
// Namespace performs the same function as databases in relational databases.
type Namespace interface {
	Start(ctx context.Context) error
	GetName() valueobjects.NamespaceName
	GetType() valueobjects.NamespaceType
	Rename(name valueobjects.NamespaceName) chan error
	SetMaxSimilarProfiles(limit uint)
	GetMaxSimilarProfiles() uint
	SetDislikeFactor(value float32)
	Stop()
}
