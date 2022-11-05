package dto

import (
	"recengine/internal/domain/services"
	"recengine/internal/domain/valueobjects"
)

// A DTO for updating a Namespace.
type NamespaceUpdateRequest struct {
	Name               string  `json:"name" binding:"required,lowercase,alphanum"`
	MaxSimilarProfiles uint    `json:"maxSimilarProfiles" binding:"omitempty,min=1"`
	DislikeFactor      float32 `json:"dislikeFactor" binding:"required,min=0,max=1"`
}

func (dto *NamespaceUpdateRequest) ToDomain() (*services.NamespaceUpdateRequest, error) {
	domainName, err := valueobjects.ParseNamespaceName(dto.Name)
	if err != nil {
		return nil, NewValidationErrorField("name", err)
	}
	domainDto := &services.NamespaceUpdateRequest{
		Name:               domainName,
		MaxSimilarProfiles: dto.MaxSimilarProfiles,
		DislikeFactor:      dto.DislikeFactor,
	}
	return domainDto, nil
}
