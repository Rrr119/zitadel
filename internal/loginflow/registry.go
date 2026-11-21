package loginflow

import (
	"fmt"
	"sync"
)

// Registry is an in-memory ConditionProvider. Conditions are keyed by
// organization and action ID, which allows organizations to orchestrate their
// login flow independently. It is safe for concurrent use and intended as the
// default provider as well as a building block for persistent providers.
type Registry struct {
	mu         sync.RWMutex
	conditions map[string]map[string]*Condition
}

// NewRegistry creates an empty condition registry.
func NewRegistry() *Registry {
	return &Registry{
		conditions: make(map[string]map[string]*Condition),
	}
}

// SetCondition registers or replaces the condition of an action within an
// organization. A nil condition removes any previously stored condition, so
// the action falls back to the unconditional legacy behavior.
func (r *Registry) SetCondition(orgID, actionID string, condition *Condition) error {
	if orgID == "" {
		return fmt.Errorf("organization ID is required")
	}
	if actionID == "" {
		return fmt.Errorf("action ID is required")
	}
	if err := condition.Validate(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if condition == nil || condition.IsEmpty() {
		if actions, ok := r.conditions[orgID]; ok {
			delete(actions, actionID)
			if len(actions) == 0 {
				delete(r.conditions, orgID)
			}
		}
		return nil
	}
	actions, ok := r.conditions[orgID]
	if !ok {
		actions = make(map[string]*Condition)
		r.conditions[orgID] = actions
	}
	actions[actionID] = condition
	return nil
}

// ConditionForAction returns the stored condition for the action in the given
// organization, or nil when none is configured.
func (r *Registry) ConditionForAction(orgID, actionID string) (*Condition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	actions, ok := r.conditions[orgID]
	if !ok {
		return nil, nil
	}
	return actions[actionID], nil
}

// RemoveOrganization removes every condition of the given organization.
func (r *Registry) RemoveOrganization(orgID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.conditions, orgID)
}
