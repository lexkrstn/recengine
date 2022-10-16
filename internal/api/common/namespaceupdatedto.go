package common

import (
	"recengine/internal/entities"
	"recengine/internal/valueobjects"
)

// A DTO for updating a Namespace.
type NamespaceUpdateDto struct {
	Name               string  `json:"name" binding:"required,lowercase,alphanum"`
	MaxSimilarProfiles uint    `json:"maxSimilarProfiles" binding:"omitempty,min=1"`
	DislikeFactor      float32 `json:"dislikeFactor" binding:"required,min=0,max=1"`
}

func (dto *NamespaceUpdateDto) ToDomain() (*entities.NamespaceUpdateDto, error) {
	domainName, err := valueobjects.ParseNamespaceName(dto.Name)
	if err != nil {
		return nil, NewValidationErrorField("name", err)
	}
	domainDto := &entities.NamespaceUpdateDto{
		Name:               domainName,
		MaxSimilarProfiles: dto.MaxSimilarProfiles,
		DislikeFactor:      dto.DislikeFactor,
	}
	return domainDto, nil
}
