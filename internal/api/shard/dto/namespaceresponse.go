package dto

import "recengine/internal/domain"

type NamespaceResponse struct {
	Name               string `json:"name"`
	Type               string `json:"type"`
	MaxSimilarProfiles uint   `json:"maxSimilarProfiles"`
}

func NewNamespaceResponse(ns domain.Namespace) *NamespaceResponse {
	return &NamespaceResponse{
		Name:               ns.GetName().Value(),
		Type:               ns.GetType().Value(),
		MaxSimilarProfiles: ns.GetMaxSimilarProfiles(),
	}
}

func MakeNamespaceResponseArray(namespaces []domain.Namespace) []NamespaceResponse {
	responses := make([]NamespaceResponse, len(namespaces))
	for i := range namespaces {
		responses[i] = *NewNamespaceResponse(namespaces[i])
	}
	return responses
}
