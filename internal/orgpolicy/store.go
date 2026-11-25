package orgpolicy

import (
	"sort"
	"sync"
	"time"
)

type orgData struct {
	organization Organization
	// branding is nil while the organization inherits the instance default.
	branding *Branding
	// loginPolicy is nil while the organization inherits the instance default.
	loginPolicy *LoginPolicy
}

type instanceData struct {
	id              string
	defaultBranding Branding
	defaultLogin    LoginPolicy
	orgs            map[string]*orgData
}

// Store is a concurrency safe, in-memory Repository. All data is separated
// by instance id and, within an instance, by organization id.
type Store struct {
	mu        sync.RWMutex
	instances map[string]*instanceData
	events    []AuditEvent
	now       func() time.Time
}

// NewStore creates an empty store.
func NewStore() *Store {
	return &Store{
		instances: make(map[string]*instanceData),
		now:       time.Now,
	}
}

// Compile time proof that Store is a Repository.
var _ Repository = (*Store)(nil)

func (s *Store) instanceLocked(id string) *instanceData {
	data, ok := s.instances[id]
	if !ok {
		data = &instanceData{
			id:              id,
			defaultBranding: DefaultBranding(),
			defaultLogin:    DefaultLoginPolicy(),
			orgs:            make(map[string]*orgData),
		}
		s.instances[id] = data
	}
	return data
}

// existingInstanceRLocked returns an already initialized instance without
// creating it, so read locked callers never mutate the instances map.
func (s *Store) existingInstanceRLocked(id string) (*instanceData, bool) {
	data, ok := s.instances[id]
	return data, ok
}

// EnsureInstance implements Repository.
func (s *Store) EnsureInstance(instanceID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.instanceLocked(instanceID)
}

// InsertOrganization implements Repository.
func (s *Store) InsertOrganization(org Organization) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	instance := s.instanceLocked(org.InstanceID)
	if _, exists := instance.orgs[org.ID]; exists {
		return ErrOrganizationExists
	}
	if org.CreatedAt.IsZero() {
		org.CreatedAt = s.now().UTC()
	}
	instance.orgs[org.ID] = &orgData{organization: org}
	return nil
}

// FindOrganization implements Repository.
func (s *Store) FindOrganization(instanceID, orgID string) (Organization, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	instance, ok := s.existingInstanceRLocked(instanceID)
	if !ok {
		return Organization{}, ErrOrganizationNotFound
	}
	org, ok := instance.orgs[orgID]
	if !ok {
		return Organization{}, ErrOrganizationNotFound
	}
	return org.organization, nil
}

// ListOrganizations implements Repository.
func (s *Store) ListOrganizations(instanceID string) []Organization {
	s.mu.RLock()
	defer s.mu.RUnlock()
	instance, ok := s.existingInstanceRLocked(instanceID)
	if !ok {
		return []Organization{}
	}
	ids := make([]string, 0, len(instance.orgs))
	for id := range instance.orgs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	result := make([]Organization, 0, len(ids))
	for _, id := range ids {
		result = append(result, instance.orgs[id].organization)
	}
	return result
}

// GetDefaultBranding implements Repository.
func (s *Store) GetDefaultBranding(instanceID string) Branding {
	s.mu.RLock()
	defer s.mu.RUnlock()
	instance, ok := s.existingInstanceRLocked(instanceID)
	if !ok {
		return DefaultBranding()
	}
	return instance.defaultBranding.Clone()
}

// SetDefaultBranding implements Repository.
func (s *Store) SetDefaultBranding(instanceID string, branding Branding) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.instanceLocked(instanceID).defaultBranding = branding.Clone()
}

// GetDefaultLoginPolicy implements Repository.
func (s *Store) GetDefaultLoginPolicy(instanceID string) LoginPolicy {
	s.mu.RLock()
	defer s.mu.RUnlock()
	instance, ok := s.existingInstanceRLocked(instanceID)
	if !ok {
		return DefaultLoginPolicy()
	}
	return instance.defaultLogin.Clone()
}

// SetDefaultLoginPolicy implements Repository.
func (s *Store) SetDefaultLoginPolicy(instanceID string, policy LoginPolicy) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.instanceLocked(instanceID).defaultLogin = policy.Clone()
}

func (s *Store) orgForUpdateLocked(instanceID, orgID string) (*orgData, error) {
	if orgID == "" {
		return nil, ErrOrganizationNotFound
	}
	org, ok := s.instanceLocked(instanceID).orgs[orgID]
	if !ok {
		return nil, ErrOrganizationNotFound
	}
	return org, nil
}

// GetOrgBranding implements Repository.
func (s *Store) GetOrgBranding(instanceID, orgID string) (Branding, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	instance, ok := s.existingInstanceRLocked(instanceID)
	if !ok {
		return Branding{}, false, ErrOrganizationNotFound
	}
	org, exists := instance.orgs[orgID]
	if !exists {
		return Branding{}, false, ErrOrganizationNotFound
	}
	if org.branding == nil {
		return Branding{}, false, nil
	}
	return org.branding.Clone(), true, nil
}

// SetOrgBranding implements Repository.
func (s *Store) SetOrgBranding(instanceID, orgID string, branding Branding) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	org, err := s.orgForUpdateLocked(instanceID, orgID)
	if err != nil {
		return err
	}
	clone := branding.Clone()
	org.branding = &clone
	return nil
}

// ClearOrgBranding implements Repository.
func (s *Store) ClearOrgBranding(instanceID, orgID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	org, err := s.orgForUpdateLocked(instanceID, orgID)
	if err != nil {
		return err
	}
	org.branding = nil
	return nil
}

// GetOrgLoginPolicy implements Repository.
func (s *Store) GetOrgLoginPolicy(instanceID, orgID string) (LoginPolicy, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	instance, ok := s.existingInstanceRLocked(instanceID)
	if !ok {
		return LoginPolicy{}, false, ErrOrganizationNotFound
	}
	org, exists := instance.orgs[orgID]
	if !exists {
		return LoginPolicy{}, false, ErrOrganizationNotFound
	}
	if org.loginPolicy == nil {
		return LoginPolicy{}, false, nil
	}
	return org.loginPolicy.Clone(), true, nil
}

// SetOrgLoginPolicy implements Repository.
func (s *Store) SetOrgLoginPolicy(instanceID, orgID string, policy LoginPolicy) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	org, err := s.orgForUpdateLocked(instanceID, orgID)
	if err != nil {
		return err
	}
	clone := policy.Clone()
	org.loginPolicy = &clone
	return nil
}

// ClearOrgLoginPolicy implements Repository.
func (s *Store) ClearOrgLoginPolicy(instanceID, orgID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	org, err := s.orgForUpdateLocked(instanceID, orgID)
	if err != nil {
		return err
	}
	org.loginPolicy = nil
	return nil
}

// AppendAuditEvent implements Repository.
func (s *Store) AppendAuditEvent(event AuditEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if event.At.IsZero() {
		event.At = s.now().UTC()
	}
	s.events = append(s.events, event)
}

// ListAuditEvents implements Repository. When instanceID is non-empty the
// result is filtered to that instance.
func (s *Store) ListAuditEvents(instanceID string) []AuditEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]AuditEvent, 0, len(s.events))
	for _, event := range s.events {
		if instanceID != "" && event.InstanceID != instanceID {
			continue
		}
		result = append(result, event)
	}
	return result
}

// AuditEvents is a convenience alias for ListAuditEvents retained for
// callers that prefer the descriptive name.
func (s *Store) AuditEvents(instanceID string) []AuditEvent {
	return s.ListAuditEvents(instanceID)
}
