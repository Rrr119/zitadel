package orgpolicy

import "context"

// PolicySource tells where a resolved policy came from.
type PolicySource string

const (
	// SourceInstance marks a policy inherited from the instance default.
	SourceInstance PolicySource = "instance_default"
	// SourceOrganization marks a policy explicitly customized by the org.
	SourceOrganization PolicySource = "organization"
)

// Resolved is an effective policy together with its inheritance provenance.
type Resolved[T any] struct {
	// Value is the effective policy, safe for the caller to mutate.
	Value T
	// Inherited is true when the value comes from the instance default.
	Inherited bool
	// Source is either SourceInstance or SourceOrganization.
	Source PolicySource
}

// OrganizationPolicySet is the resolved branding and login policy of one
// organization, the unit used by administrative overviews and tests that
// verify multi organization isolation.
type OrganizationPolicySet struct {
	Organization Organization
	Branding     Resolved[Branding]
	LoginPolicy  Resolved[LoginPolicy]
}

// Service exposes organization scoped branding and login policy operations
// for one IAM instance. All operations are isolated to the bound instance.
type Service struct {
	repo       Repository
	instanceID string
}

// NewService builds a policy service for instanceID backed by repo. When
// repo is nil, the concurrency safe in-memory Store is used.
func NewService(repo Repository, instanceID string) (*Service, error) {
	if instanceID == "" {
		return nil, ErrInvalidInstanceID
	}
	if repo == nil {
		repo = NewStore()
	}
	repo.EnsureInstance(instanceID)
	return &Service{repo: repo, instanceID: instanceID}, nil
}

// InstanceID returns the instance the service is bound to.
func (s *Service) InstanceID() string { return s.instanceID }

// Repository exposes the configured persistence seam for wiring/inspection.
func (s *Service) Repository() Repository { return s.repo }

func (s *Service) audit(orgID string, scope PolicyScope, action AuditAction) {
	s.repo.AppendAuditEvent(AuditEvent{
		InstanceID:     s.instanceID,
		OrganizationID: orgID,
		Scope:          scope,
		Action:         action,
	})
}

// RegisterOrganization adds an organization to the instance.
func (s *Service) RegisterOrganization(_ context.Context, orgID, name string) (Organization, error) {
	if orgID == "" {
		return Organization{}, ErrOrganizationNotFound
	}
	org := Organization{ID: orgID, Name: name, InstanceID: s.instanceID}
	if err := s.repo.InsertOrganization(org); err != nil {
		return Organization{}, err
	}
	stored, err := s.repo.FindOrganization(s.instanceID, orgID)
	if err != nil {
		return Organization{}, err
	}
	s.audit(orgID, ScopeOrganization, ActionRegistered)
	return stored, nil
}

// ListOrganizations returns the registered organizations ordered by id.
func (s *Service) ListOrganizations(_ context.Context) []Organization {
	return s.repo.ListOrganizations(s.instanceID)
}

// GetDefaultBranding returns the instance default brand.
func (s *Service) GetDefaultBranding(_ context.Context) Branding {
	return s.repo.GetDefaultBranding(s.instanceID)
}

// UpdateDefaultBranding replaces the instance default brand.
func (s *Service) UpdateDefaultBranding(_ context.Context, branding Branding) error {
	if err := branding.Validate(); err != nil {
		return err
	}
	s.repo.SetDefaultBranding(s.instanceID, branding)
	s.audit("", ScopeBranding, ActionUpdatedDefault)
	return nil
}

// GetDefaultLoginPolicy returns the instance default login policy.
func (s *Service) GetDefaultLoginPolicy(_ context.Context) LoginPolicy {
	return s.repo.GetDefaultLoginPolicy(s.instanceID)
}

// UpdateDefaultLoginPolicy replaces the instance default login policy.
func (s *Service) UpdateDefaultLoginPolicy(_ context.Context, policy LoginPolicy) error {
	if err := policy.Validate(); err != nil {
		return err
	}
	s.repo.SetDefaultLoginPolicy(s.instanceID, policy)
	s.audit("", ScopeLoginPolicy, ActionUpdatedDefault)
	return nil
}

// GetBranding resolves the effective brand of an organization: the org
// customization when present, otherwise a copy of the instance default.
func (s *Service) GetBranding(_ context.Context, orgID string) (Resolved[Branding], error) {
	branding, customized, err := s.repo.GetOrgBranding(s.instanceID, orgID)
	if err != nil {
		return Resolved[Branding]{}, err
	}
	if customized {
		return Resolved[Branding]{Value: branding, Inherited: false, Source: SourceOrganization}, nil
	}
	return Resolved[Branding]{
		Value:     s.repo.GetDefaultBranding(s.instanceID),
		Inherited: true,
		Source:    SourceInstance,
	}, nil
}

// UpdateBranding sets an explicit, validated brand for the organization.
func (s *Service) UpdateBranding(_ context.Context, orgID string, branding Branding) error {
	if err := branding.Validate(); err != nil {
		return err
	}
	if err := s.repo.SetOrgBranding(s.instanceID, orgID, branding); err != nil {
		return err
	}
	s.audit(orgID, ScopeBranding, ActionUpdatedOrg)
	return nil
}

// PatchBranding merges the patch over the currently effective brand and
// stores the result as an organization customization. Unset fields keep
// inheriting their current (often instance default) value.
func (s *Service) PatchBranding(ctx context.Context, orgID string, patch BrandingPatch) (Branding, error) {
	current, err := s.GetBranding(ctx, orgID)
	if err != nil {
		return Branding{}, err
	}
	merged := current.Value.Apply(patch)
	if err := s.UpdateBranding(ctx, orgID, merged); err != nil {
		return Branding{}, err
	}
	return merged, nil
}

// ResetBranding removes the organization brand so it inherits again.
func (s *Service) ResetBranding(_ context.Context, orgID string) error {
	if err := s.repo.ClearOrgBranding(s.instanceID, orgID); err != nil {
		return err
	}
	s.audit(orgID, ScopeBranding, ActionResetOrg)
	return nil
}

// GetLoginPolicy resolves the effective login policy of an organization.
func (s *Service) GetLoginPolicy(_ context.Context, orgID string) (Resolved[LoginPolicy], error) {
	policy, customized, err := s.repo.GetOrgLoginPolicy(s.instanceID, orgID)
	if err != nil {
		return Resolved[LoginPolicy]{}, err
	}
	if customized {
		return Resolved[LoginPolicy]{Value: policy, Inherited: false, Source: SourceOrganization}, nil
	}
	return Resolved[LoginPolicy]{
		Value:     s.repo.GetDefaultLoginPolicy(s.instanceID),
		Inherited: true,
		Source:    SourceInstance,
	}, nil
}

// UpdateLoginPolicy sets an explicit, validated login policy for the org.
func (s *Service) UpdateLoginPolicy(_ context.Context, orgID string, policy LoginPolicy) error {
	if err := policy.Validate(); err != nil {
		return err
	}
	if err := s.repo.SetOrgLoginPolicy(s.instanceID, orgID, policy); err != nil {
		return err
	}
	s.audit(orgID, ScopeLoginPolicy, ActionUpdatedOrg)
	return nil
}

// PatchLoginPolicy merges the patch over the effective policy and stores it.
func (s *Service) PatchLoginPolicy(ctx context.Context, orgID string, patch LoginPolicyPatch) (LoginPolicy, error) {
	current, err := s.GetLoginPolicy(ctx, orgID)
	if err != nil {
		return LoginPolicy{}, err
	}
	merged := current.Value.Apply(patch)
	if err := s.UpdateLoginPolicy(ctx, orgID, merged); err != nil {
		return LoginPolicy{}, err
	}
	return merged, nil
}

// ResetLoginPolicy removes the organization policy so it inherits again.
func (s *Service) ResetLoginPolicy(_ context.Context, orgID string) error {
	if err := s.repo.ClearOrgLoginPolicy(s.instanceID, orgID); err != nil {
		return err
	}
	s.audit(orgID, ScopeLoginPolicy, ActionResetOrg)
	return nil
}

// GetOrganizationPolicySet resolves both policies for one organization.
func (s *Service) GetOrganizationPolicySet(ctx context.Context, orgID string) (*OrganizationPolicySet, error) {
	organization, err := s.repo.FindOrganization(s.instanceID, orgID)
	if err != nil {
		return nil, err
	}
	branding, err := s.GetBranding(ctx, orgID)
	if err != nil {
		return nil, err
	}
	loginPolicy, err := s.GetLoginPolicy(ctx, orgID)
	if err != nil {
		return nil, err
	}
	return &OrganizationPolicySet{
		Organization: organization,
		Branding:     branding,
		LoginPolicy:  loginPolicy,
	}, nil
}

// ListOrganizationPolicySets resolves every organization on the instance.
func (s *Service) ListOrganizationPolicySets(ctx context.Context) ([]*OrganizationPolicySet, error) {
	organizations := s.ListOrganizations(ctx)
	result := make([]*OrganizationPolicySet, 0, len(organizations))
	for _, org := range organizations {
		set, err := s.GetOrganizationPolicySet(ctx, org.ID)
		if err != nil {
			return nil, err
		}
		result = append(result, set)
	}
	return result, nil
}
