package entities

import (
	"context"
	"recengine/internal/valueobjects"
)

// Interface that domains of any type must implement.
type Namespace interface {
	Start(ctx context.Context)
	GetName() valueobjects.NamespaceName
	GetType() valueobjects.NamespaceType
	Rename(name valueobjects.NamespaceName) chan error
	SetMaxSimilarProfiles(limit uint)
	GetMaxSimilarProfiles() uint
	SetDislikeFactor(value float32)
	Stop()
}
