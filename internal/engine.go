package recengine

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"recengine/internal/helpers"
)

// Recommendation engine instance.
type Engine struct {
	domains         []Domain
	domainsFilePath string
}

// Creates an Engine instance.
func NewEngine() *Engine {
	engine := Engine{
		domains:         []Domain{},
		domainsFilePath: "./domains.json",
	}
	engine.LoadDomains()
	return &engine
}

// Starts all domains to run their jobs on separate threads.
func (engine *Engine) Start(ctx context.Context) error {
	for _, domain := range engine.domains {
		domain.Start(ctx)
	}
	return nil
}

// Loads domain list from the file.
// Warning the function is not thread-safe, so must be called only before
// starting the engine.
func (engine *Engine) LoadDomains() error {
	data, err := os.ReadFile(engine.domainsFilePath)
	if err != nil {
		return fmt.Errorf("Failed to open %s: %v", engine.domainsFilePath, err)
	}
	err = json.Unmarshal(data, &engine.domains)
	if err != nil {
		return fmt.Errorf("Failed to decode %s: %v", engine.domainsFilePath, err)
	}
	return nil
}

// Saves domain list to the file.
func (engine *Engine) SaveDomains() error {
	data, err := json.Marshal(engine.domains)
	if err != nil {
		return fmt.Errorf("Failed to encode domains: %v", err)
	}
	err = os.WriteFile(engine.domainsFilePath, data, 0644)
	if err != nil {
		return fmt.Errorf("Failed to write to %s: %v", engine.domainsFilePath, err)
	}
	return nil
}

// Adds domain registration to the engine and persists the change.
func (engine *Engine) AddDomain(dto *DomainCreateInput) (Domain, error) {
	if engine.getDomainIndexByName(dto.Name) >= 0 {
		return nil, fmt.Errorf("Domain name %s is already taken", dto.Name)
	}
	domain, err := NewDomain(dto)
	if err != nil {
		return nil, err
	}
	engine.domains = append(engine.domains, domain)
	if err = engine.SaveDomains(); err != nil {
		return nil, err
	}
	return domain, nil
}

// Updates the domain by it's name and persists the change.
func (engine *Engine) UpdateDomain(name string, dto *DomainUpdateInput) (Domain, error) {
	domain := engine.GetDomainByName(name)
	if domain == nil {
		return nil, fmt.Errorf("Domain %s not found", name)
	}
	if domain.GetName() != dto.Name && engine.getDomainIndexByName(dto.Name) >= 0 {
		return nil, fmt.Errorf("Domain name %s is already taken", dto.Name)
	}
	if err := <-domain.Rename(dto.Name); err != nil {
		return nil, err
	}
	domain.SetDislikeFactor(dto.DislikeFactor)
	domain.SetMaxSimilarProfiles(dto.MaxSimilarProfiles)
	if err := engine.SaveDomains(); err != nil {
		return nil, err
	}
	return domain, nil
}

// Returns the pointer to the domain by its name, or nil if not found.
func (engine *Engine) GetDomainByName(name string) Domain {
	index := engine.getDomainIndexByName(name)
	if index < 0 {
		return nil
	}
	return engine.domains[index]
}

// Returns the index of the domain in the domains list.
func (engine *Engine) getDomainIndexByName(name string) int {
	for i, domain := range engine.domains {
		if domain.GetName() == name {
			return i
		}
	}
	return -1
}

// Removes domain registration from the engine and persists the change.
// The deleted domain stops running automatically.
func (engine *Engine) DeleteDomain(name string) error {
	index := engine.getDomainIndexByName(name)
	if index < 0 {
		return fmt.Errorf("No domain %s", name)
	}
	engine.domains[index].Stop()
	engine.domains = helpers.Remove(engine.domains, index)
	if err := engine.SaveDomains(); err != nil {
		return err
	}
	return nil
}
