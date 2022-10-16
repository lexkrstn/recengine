package common

import (
	"recengine/internal/entities"
	"recengine/internal/valueobjects"
)

// A DTO for creating a Namespace.
type NamespaceCreateDto struct {
	Name               string  `json:"name" binding:"required,lowercase,alphanum"`
	Type               string  `json:"type" binding:"required,oneof=like"`
	MaxSimilarProfiles uint    `json:"maxSimilarProfiles" binding:"omitempty,min=1"`
	DislikeFactor      float32 `json:"dislikeFactor" binding:"required,min=0,max=1"`
}

func (dto *NamespaceCreateDto) ToDomain() (*entities.NamespaceCreateDto, error) {
	var ve *ValidationError
	domainName, err := valueobjects.ParseNamespaceName(dto.Name)
	if err != nil {
		ve = AddValidationErrorField(ve, "name", err)
	}
	domainType, err := valueobjects.ParseNamespaceType(dto.Type)
	if err != nil {
		ve = AddValidationErrorField(ve, "type", err)
	}
	if ve != nil {
		return nil, ve
	}
	domainDto := &entities.NamespaceCreateDto{
		Name:               domainName,
		Type:               domainType,
		MaxSimilarProfiles: dto.MaxSimilarProfiles,
		DislikeFactor:      dto.DislikeFactor,
	}
	return domainDto, nil
}
