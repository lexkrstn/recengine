package entities

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"recengine/internal/helpers"
	"recengine/internal/valueobjects"
)

// A DTO for creating a Namespace.
type NamespaceCreateDto struct {
	Name               valueobjects.NamespaceName
	Type               valueobjects.NamespaceType
	MaxSimilarProfiles uint
	DislikeFactor      float32
}

// A DTO for updating a Namespace.
type NamespaceUpdateDto struct {
	Name               valueobjects.NamespaceName
	MaxSimilarProfiles uint
	DislikeFactor      float32
}

// Manages namespaces.
type NamespaceService struct {
	namespaces []Namespace
	context    context.Context
	basePath   string
}

// Creates a NamespaceService.
func NewNamespaceService(context context.Context) *NamespaceService {
	basePath := os.Getenv("REC_PATH")
	if basePath != "" && basePath[len(basePath)-1] != '/' {
		basePath = basePath + "/"
	}
	return &NamespaceService{
		namespaces: make([]Namespace, 0),
		context:    context,
		basePath:   basePath,
	}
}

// Namespace's factory function.
func (s *NamespaceService) forgeNamespace(dto *NamespaceCreateDto) (Namespace, error) {
	switch dto.Type.Value() {
	case valueobjects.NamespaceTypeLike:
		ns := NewLikeNamespace(&LikeNamespaceDto{
			Name:               dto.Name,
			MaxSimilarProfiles: dto.MaxSimilarProfiles,
			DislikeFactor:      dto.DislikeFactor,
		})
		return ns, nil
	default:
		return nil, fmt.Errorf("Unknown domain type %s", dto.Type)
	}
}

// Starts all namespaces to run their jobs on separate threads.
func (s *NamespaceService) Start(ctx context.Context) error {
	for _, domain := range s.namespaces {
		domain.Start(ctx)
	}
	return nil
}

func (s *NamespaceService) getNamespacesJsonPath() string {
	return s.basePath + "namespaces.json"
}

// Loads namespace list from the file.
// Warning the function is not thread-safe, so must be called only before
// starting the engine.
func (s *NamespaceService) LoadNamespaces() error {
	filePath := s.getNamespacesJsonPath()
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("Failed to open %s: %v", filePath, err)
	}
	err = json.Unmarshal(data, &s.namespaces)
	if err != nil {
		return fmt.Errorf("Failed to decode %s: %v", filePath, err)
	}
	return nil
}

// Saves namespace list to the file.
func (s *NamespaceService) SaveNamespaces() error {
	data, err := json.Marshal(s.namespaces)
	if err != nil {
		return fmt.Errorf("Failed to encode namespaces: %v", err)
	}
	filePath := s.getNamespacesJsonPath()
	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		return fmt.Errorf("Failed to write to %s: %v", filePath, err)
	}
	return nil
}

// Returns the list of currently loaded namespaces.
func (s *NamespaceService) GetNamespaces() []Namespace {
	return s.namespaces
}

// Returns the index of the namespace in the namespace list.
func (s *NamespaceService) getNamespaceIndexByName(name valueobjects.NamespaceName) int {
	for i, ns := range s.namespaces {
		if ns.GetName() == name {
			return i
		}
	}
	return -1
}

// Returns the pointer to the namespace by its name, or nil if not found.
func (s *NamespaceService) GetNamespaceByName(name valueobjects.NamespaceName) Namespace {
	index := s.getNamespaceIndexByName(name)
	if index < 0 {
		return nil
	}
	return s.namespaces[index]
}

// Adds domain registration to the engine and persists the change.
func (s *NamespaceService) CreateNamespace(dto *NamespaceCreateDto) (Namespace, error) {
	if s.getNamespaceIndexByName(dto.Name) >= 0 {
		return nil, fmt.Errorf("Namespace name %s is already taken", dto.Name)
	}
	ns, err := s.forgeNamespace(dto)
	if err != nil {
		return nil, err
	}
	s.namespaces = append(s.namespaces, ns)
	if err = s.SaveNamespaces(); err != nil {
		return nil, err
	}
	return ns, nil
}

// Updates the namespace by it's name and persists the change.
func (s *NamespaceService) UpdateNamespace(
	name valueobjects.NamespaceName,
	dto *NamespaceUpdateDto,
) (Namespace, error) {
	ns := s.GetNamespaceByName(name)
	if ns == nil {
		return nil, fmt.Errorf("Namespace %s not found", name)
	}
	if ns.GetName() != dto.Name && s.getNamespaceIndexByName(dto.Name) >= 0 {
		return nil, fmt.Errorf("Namespace name %s is already taken", dto.Name)
	}
	if err := <-ns.Rename(dto.Name); err != nil {
		return nil, err
	}
	ns.SetDislikeFactor(dto.DislikeFactor)
	ns.SetMaxSimilarProfiles(dto.MaxSimilarProfiles)
	if err := s.SaveNamespaces(); err != nil {
		return nil, err
	}
	return ns, nil
}

// Removes namespace registration from the engine and persists the change.
// The deleted namespace stops running automatically.
func (s *NamespaceService) DeleteNamespace(name valueobjects.NamespaceName) error {
	index := s.getNamespaceIndexByName(name)
	if index < 0 {
		return fmt.Errorf("No namespace %s", name)
	}
	s.namespaces[index].Stop()
	s.namespaces = helpers.Remove(s.namespaces, index)
	if err := s.SaveNamespaces(); err != nil {
		return err
	}
	return nil
}
