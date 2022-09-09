package recengine

import (
	"context"
	"fmt"
	"recengine/internal/like"
)

// A DTO for creating a Domain.
type DomainCreateInput struct {
	Name               string  `json:"name" binding:"required,lowercase,alphanum"`
	Type               string  `json:"type" binding:"required,oneof=like"`
	MaxSimilarProfiles int     `json:"maxSimilarProfiles" binding:"omitempty,min=1"`
	DislikeFactor      float32 `json:"dislikeFactor" binding:"required,min=0,max=1"`
}

// A DTO for updating a Domain.
type DomainUpdateInput = struct {
	Name               string  `json:"name" binding:"required,lowercase,alphanum"`
	MaxSimilarProfiles int     `json:"maxSimilarProfiles" binding:"omitempty,min=1"`
	DislikeFactor      float32 `json:"dislikeFactor" binding:"required,min=0,max=1"`
}

// Interface that domains of any type must implement.
type Domain interface {
	Start(ctx context.Context)
	GetName() string
	Rename(name string) chan error
	SetMaxSimilarProfiles(value int)
	SetDislikeFactor(value float32)
	Stop()
}

// Domain's factory function.
func NewDomain(dto *DomainCreateInput) (Domain, error) {
	switch dto.Type {
	case "like":
		domain := like.NewDomain(&like.DomainCreateInput{
			Name:               dto.Name,
			MaxSimilarProfiles: dto.MaxSimilarProfiles,
			DislikeFactor:      dto.DislikeFactor,
		})
		return domain, nil
	default:
		return nil, fmt.Errorf("Unknown domain type %s", dto.Type)
	}
}
