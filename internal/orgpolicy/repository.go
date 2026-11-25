package orgpolicy

import "time"

// PolicyScope identifies which policy kind an audit event concerns.
type PolicyScope string

const (
	ScopeBranding     PolicyScope = "branding"
	ScopeLoginPolicy  PolicyScope = "login_policy"
	ScopeOrganization PolicyScope = "organization"
)

// AuditAction describes the mutation applied to a policy.
type AuditAction string

const (
	ActionRegistered     AuditAction = "registered"
	ActionUpdatedDefault AuditAction = "updated_default"
	ActionUpdatedOrg     AuditAction = "updated_org"
	ActionResetOrg       AuditAction = "reset_org"
)

// AuditEvent is a minimal append-only audit record. ZITADEL persists richer
// events in its event store; this lightweight log keeps the package
// observable and testable without a database.
type AuditEvent struct {
	At             time.Time
	InstanceID     string
	OrganizationID string
	Scope          PolicyScope
	Action         AuditAction
}

// Organization is a registered tenant organization inside an instance.
type Organization struct {
	ID         string
	Name       string
	InstanceID string
	CreatedAt  time.Time
}

// Repository is the persistence seam for organization policies.
//
// Store provides a concurrency safe in-memory reference implementation. A
// production ZITADEL adapter would implement this interface in the
// command/query or repository layer backed by the relational tables (and
// keep writing the corresponding audit events), without changing Service.
//
// Every method is scoped by instanceID and, where applicable, orgID, so
// implementations must preserve instance and organization isolation.
type Repository interface {
	// EnsureInstance creates the instance bucket with built-in defaults when
	// it is observed for the first time.
	EnsureInstance(instanceID string)

	InsertOrganization(org Organization) error
	FindOrganization(instanceID, orgID string) (Organization, error)
	ListOrganizations(instanceID string) []Organization

	GetDefaultBranding(instanceID string) Branding
	SetDefaultBranding(instanceID string, branding Branding)
	GetDefaultLoginPolicy(instanceID string) LoginPolicy
	SetDefaultLoginPolicy(instanceID string, policy LoginPolicy)

	// GetOrgBranding returns customized=false when the organization has no
	// own brand and must inherit the instance default. It returns
	// ErrOrganizationNotFound when the organization does not exist.
	GetOrgBranding(instanceID, orgID string) (branding Branding, customized bool, err error)
	SetOrgBranding(instanceID, orgID string, branding Branding) error
	ClearOrgBranding(instanceID, orgID string) error

	GetOrgLoginPolicy(instanceID, orgID string) (policy LoginPolicy, customized bool, err error)
	SetOrgLoginPolicy(instanceID, orgID string, policy LoginPolicy) error
	ClearOrgLoginPolicy(instanceID, orgID string) error

	AppendAuditEvent(event AuditEvent)
	ListAuditEvents(instanceID string) []AuditEvent
}
