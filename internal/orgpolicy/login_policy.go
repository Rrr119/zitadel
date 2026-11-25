package orgpolicy

import "time"

// SecondFactorType enumerates supported second factors. The numeric values
// are aligned with domain.SecondFactorType for lossless iamcompat mapping.
type SecondFactorType int32

const (
	SecondFactorUnspecified SecondFactorType = iota
	SecondFactorTOTP
	SecondFactorU2F
	SecondFactorOTPEmail
	SecondFactorOTPSMS
)

// MultiFactorType enumerates supported multi factors.
type MultiFactorType int32

const (
	MultiFactorUnspecified MultiFactorType = iota
	MultiFactorU2FWithPIN
)

// PasswordlessType controls whether passwordless login is allowed.
type PasswordlessType int32

const (
	PasswordlessNotAllowed PasswordlessType = iota
	PasswordlessAllowed
)

// LoginPolicy is the resolved authentication policy of an organization or
// the instance. It mirrors the effective subset of domain.LoginPolicy.
type LoginPolicy struct {
	AllowUsernamePassword bool `json:"allowUsernamePassword"`
	AllowRegister         bool `json:"allowRegister"`
	AllowExternalIDP      bool `json:"allowExternalIdp"`

	ForceMFA          bool `json:"forceMfa"`
	ForceMFALocalOnly bool `json:"forceMfaLocalOnly"`

	HidePasswordReset      bool `json:"hidePasswordReset"`
	IgnoreUnknownUsernames bool `json:"ignoreUnknownUsernames"`
	AllowDomainDiscovery   bool `json:"allowDomainDiscovery"`

	DisableLoginWithEmail bool `json:"disableLoginWithEmail"`
	DisableLoginWithPhone bool `json:"disableLoginWithPhone"`

	PasswordlessType PasswordlessType `json:"passwordlessType"`

	SecondFactors []SecondFactorType `json:"secondFactors,omitempty"`
	MultiFactors  []MultiFactorType  `json:"multiFactors,omitempty"`

	DefaultRedirectURI string `json:"defaultRedirectUri,omitempty"`

	PasswordCheckLifetime      time.Duration `json:"passwordCheckLifetime,omitempty"`
	SecondFactorCheckLifetime  time.Duration `json:"secondFactorCheckLifetime,omitempty"`
	MultiFactorCheckLifetime   time.Duration `json:"multiFactorCheckLifetime,omitempty"`
	MFAInitSkipLifetime        time.Duration `json:"mfaInitSkipLifetime,omitempty"`
	ExternalLoginCheckLifetime time.Duration `json:"externalLoginCheckLifetime,omitempty"`
}

// LoginPolicyPatch carries optional organization overrides. Nil scalar
// pointers inherit the effective value. A nil factor slice leaves the factor
// lists untouched; a non-nil slice (including an empty one) replaces them.
type LoginPolicyPatch struct {
	AllowUsernamePassword *bool `json:"allowUsernamePassword,omitempty"`
	AllowRegister         *bool `json:"allowRegister,omitempty"`
	AllowExternalIDP      *bool `json:"allowExternalIdp,omitempty"`

	ForceMFA          *bool `json:"forceMfa,omitempty"`
	ForceMFALocalOnly *bool `json:"forceMfaLocalOnly,omitempty"`

	HidePasswordReset      *bool `json:"hidePasswordReset,omitempty"`
	IgnoreUnknownUsernames *bool `json:"ignoreUnknownUsernames,omitempty"`
	AllowDomainDiscovery   *bool `json:"allowDomainDiscovery,omitempty"`

	DisableLoginWithEmail *bool `json:"disableLoginWithEmail,omitempty"`
	DisableLoginWithPhone *bool `json:"disableLoginWithPhone,omitempty"`

	PasswordlessType *PasswordlessType `json:"passwordlessType,omitempty"`

	SecondFactors []SecondFactorType `json:"secondFactors,omitempty"`
	MultiFactors  []MultiFactorType  `json:"multiFactors,omitempty"`

	DefaultRedirectURI *string `json:"defaultRedirectUri,omitempty"`

	PasswordCheckLifetime      *time.Duration `json:"passwordCheckLifetime,omitempty"`
	SecondFactorCheckLifetime  *time.Duration `json:"secondFactorCheckLifetime,omitempty"`
	MultiFactorCheckLifetime   *time.Duration `json:"multiFactorCheckLifetime,omitempty"`
	MFAInitSkipLifetime        *time.Duration `json:"mfaInitSkipLifetime,omitempty"`
	ExternalLoginCheckLifetime *time.Duration `json:"externalLoginCheckLifetime,omitempty"`
}

// DefaultLoginPolicy returns the built-in permissive instance login policy.
func DefaultLoginPolicy() LoginPolicy {
	return LoginPolicy{
		AllowUsernamePassword: true,
		AllowRegister:         true,
		AllowExternalIDP:      true,
		PasswordlessType:      PasswordlessAllowed,
		SecondFactors:         []SecondFactorType{SecondFactorTOTP},
		PasswordCheckLifetime: 240 * time.Hour,
		MFAInitSkipLifetime:   30 * 24 * time.Hour,
	}
}

// Clone returns a deep copy so callers can mutate slices without affecting
// the stored policy.
func (p LoginPolicy) Clone() LoginPolicy {
	out := p
	if p.SecondFactors != nil {
		out.SecondFactors = append([]SecondFactorType(nil), p.SecondFactors...)
	}
	if p.MultiFactors != nil {
		out.MultiFactors = append([]MultiFactorType(nil), p.MultiFactors...)
	}
	return out
}

// Apply returns a clone of p with the patch applied.
func (p LoginPolicy) Apply(patch LoginPolicyPatch) LoginPolicy {
	out := p.Clone()

	applyBool := func(target *bool, value *bool) {
		if value != nil {
			*target = *value
		}
	}
	applyBool(&out.AllowUsernamePassword, patch.AllowUsernamePassword)
	applyBool(&out.AllowRegister, patch.AllowRegister)
	applyBool(&out.AllowExternalIDP, patch.AllowExternalIDP)
	applyBool(&out.ForceMFA, patch.ForceMFA)
	applyBool(&out.ForceMFALocalOnly, patch.ForceMFALocalOnly)
	applyBool(&out.HidePasswordReset, patch.HidePasswordReset)
	applyBool(&out.IgnoreUnknownUsernames, patch.IgnoreUnknownUsernames)
	applyBool(&out.AllowDomainDiscovery, patch.AllowDomainDiscovery)
	applyBool(&out.DisableLoginWithEmail, patch.DisableLoginWithEmail)
	applyBool(&out.DisableLoginWithPhone, patch.DisableLoginWithPhone)

	if patch.PasswordlessType != nil {
		out.PasswordlessType = *patch.PasswordlessType
	}
	if patch.SecondFactors != nil {
		out.SecondFactors = append([]SecondFactorType(nil), patch.SecondFactors...)
	}
	if patch.MultiFactors != nil {
		out.MultiFactors = append([]MultiFactorType(nil), patch.MultiFactors...)
	}
	if patch.DefaultRedirectURI != nil {
		out.DefaultRedirectURI = *patch.DefaultRedirectURI
	}
	if patch.PasswordCheckLifetime != nil {
		out.PasswordCheckLifetime = *patch.PasswordCheckLifetime
	}
	if patch.SecondFactorCheckLifetime != nil {
		out.SecondFactorCheckLifetime = *patch.SecondFactorCheckLifetime
	}
	if patch.MultiFactorCheckLifetime != nil {
		out.MultiFactorCheckLifetime = *patch.MultiFactorCheckLifetime
	}
	if patch.MFAInitSkipLifetime != nil {
		out.MFAInitSkipLifetime = *patch.MFAInitSkipLifetime
	}
	if patch.ExternalLoginCheckLifetime != nil {
		out.ExternalLoginCheckLifetime = *patch.ExternalLoginCheckLifetime
	}
	return out
}

// Validate checks enumerations, lifetimes and the redirect URI. It mirrors
// domain.ValidateDefaultRedirectURI semantics.
func (p LoginPolicy) Validate() error {
	var issues []FieldError

	if p.PasswordlessType < PasswordlessNotAllowed || p.PasswordlessType > PasswordlessAllowed {
		issues = append(issues, FieldError{Field: "passwordlessType", Message: "must be 0 (not allowed) or 1 (allowed)"})
	}
	for i, factor := range p.SecondFactors {
		if factor < SecondFactorTOTP || factor > SecondFactorOTPSMS {
			issues = append(issues, FieldError{Field: "secondFactors", Message: "index " + itoa(i) + " is not a supported second factor"})
		}
	}
	for i, factor := range p.MultiFactors {
		if factor != MultiFactorU2FWithPIN {
			issues = append(issues, FieldError{Field: "multiFactors", Message: "index " + itoa(i) + " is not a supported multi factor"})
		}
	}
	if p.PasswordCheckLifetime < 0 || p.SecondFactorCheckLifetime < 0 ||
		p.MultiFactorCheckLifetime < 0 || p.MFAInitSkipLifetime < 0 ||
		p.ExternalLoginCheckLifetime < 0 {
		issues = append(issues, FieldError{Field: "lifetimes", Message: "check lifetimes must not be negative"})
	}
	if p.DefaultRedirectURI != "" && !validDefaultRedirectURI(p.DefaultRedirectURI) {
		issues = append(issues, FieldError{Field: "defaultRedirectUri", Message: "must be an absolute http(s) URL or a valid custom scheme URI"})
	}
	return validationError(issues)
}

// validDefaultRedirectURI reproduces the rule used by
// domain.ValidateDefaultRedirectURI.
func validDefaultRedirectURI(raw string) bool {
	if raw == "" {
		return true
	}
	parsed, err := parseURL(raw)
	if err != nil {
		return false
	}
	switch parsed.Scheme {
	case "":
		return false
	case "http", "https":
		return parsed.Host != ""
	default:
		return true
	}
}
