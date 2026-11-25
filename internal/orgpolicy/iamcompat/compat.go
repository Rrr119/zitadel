// Package iamcompat maps the organization policy model to the canonical
// ZITADEL IAM domain types (domain.LabelPolicy and domain.LoginPolicy).
//
// It exists to prove the organization branding and inheritance model in
// package orgpolicy is wire compatible with the policies used by the
// existing event sourced command/query stack and the Management/Admin APIs.
package iamcompat

import (
	"github.com/zitadel/zitadel/internal/domain"
	"github.com/zitadel/zitadel/internal/orgpolicy"
)

// BrandingToLabelPolicy converts an orgpolicy brand into the canonical
// domain.LabelPolicy.
//
// resourceOwner is the organization id for an organization customization or
// the instance id for an instance default. inherited controls the canonical
// Default flag: a resolved inherited policy is returned with Default == true,
// exactly like ZITADEL query responses for default policies.
func BrandingToLabelPolicy(branding orgpolicy.Branding, instanceID, resourceOwner string, inherited bool) *domain.LabelPolicy {
	return &domain.LabelPolicy{
		ObjectRoot: objectRoot(instanceID, resourceOwner),
		State:      domain.LabelPolicyStateActive,
		Default:    inherited,

		PrimaryColor:    branding.Light.PrimaryColor,
		BackgroundColor: branding.Light.BackgroundColor,
		WarnColor:       branding.Light.WarnColor,
		FontColor:       branding.Light.FontColor,
		LogoURL:         branding.Light.LogoURL,
		IconURL:         branding.Light.IconURL,

		PrimaryColorDark:    branding.Dark.PrimaryColor,
		BackgroundColorDark: branding.Dark.BackgroundColor,
		WarnColorDark:       branding.Dark.WarnColor,
		FontColorDark:       branding.Dark.FontColor,
		LogoDarkURL:         branding.Dark.LogoURL,
		IconDarkURL:         branding.Dark.IconURL,

		Font: branding.FontURL,

		HideLoginNameSuffix: branding.HideLoginNameSuffix,
		ErrorMsgPopup:       branding.ErrorMsgPopup,
		DisableWatermark:    branding.DisableWatermark,
		ThemeMode:           domain.LabelPolicyThemeMode(branding.ThemeMode),
	}
}

// LabelPolicyToBranding converts a canonical domain.LabelPolicy back into an
// orgpolicy.Branding.
func LabelPolicyToBranding(policy *domain.LabelPolicy) orgpolicy.Branding {
	if policy == nil {
		return orgpolicy.DefaultBranding()
	}
	return orgpolicy.Branding{
		Light: orgpolicy.ThemeColors{
			PrimaryColor:    policy.PrimaryColor,
			BackgroundColor: policy.BackgroundColor,
			WarnColor:       policy.WarnColor,
			FontColor:       policy.FontColor,
			LogoURL:         policy.LogoURL,
			IconURL:         policy.IconURL,
		},
		Dark: orgpolicy.ThemeColors{
			PrimaryColor:    policy.PrimaryColorDark,
			BackgroundColor: policy.BackgroundColorDark,
			WarnColor:       policy.WarnColorDark,
			FontColor:       policy.FontColorDark,
			LogoURL:         policy.LogoDarkURL,
			IconURL:         policy.IconDarkURL,
		},
		FontURL:             policy.Font,
		HideLoginNameSuffix: policy.HideLoginNameSuffix,
		ErrorMsgPopup:       policy.ErrorMsgPopup,
		DisableWatermark:    policy.DisableWatermark,
		ThemeMode:           orgpolicy.ThemeMode(policy.ThemeMode),
	}
}

// LoginPolicyToDomain converts an orgpolicy login policy into the canonical
// domain.LoginPolicy, carrying ownership and the Default/inheritance flag.
func LoginPolicyToDomain(policy orgpolicy.LoginPolicy, instanceID, resourceOwner string, inherited bool) *domain.LoginPolicy {
	secondFactors := make([]domain.SecondFactorType, 0, len(policy.SecondFactors))
	for _, factor := range policy.SecondFactors {
		secondFactors = append(secondFactors, domain.SecondFactorType(factor))
	}
	multiFactors := make([]domain.MultiFactorType, 0, len(policy.MultiFactors))
	for _, factor := range policy.MultiFactors {
		multiFactors = append(multiFactors, domain.MultiFactorType(factor))
	}

	return &domain.LoginPolicy{
		ObjectRoot: objectRoot(instanceID, resourceOwner),
		Default:    inherited,

		AllowUsernamePassword:  policy.AllowUsernamePassword,
		AllowRegister:          policy.AllowRegister,
		AllowExternalIDP:       policy.AllowExternalIDP,
		ForceMFA:               policy.ForceMFA,
		ForceMFALocalOnly:      policy.ForceMFALocalOnly,
		SecondFactors:          secondFactors,
		MultiFactors:           multiFactors,
		PasswordlessType:       domain.PasswordlessType(policy.PasswordlessType),
		HidePasswordReset:      policy.HidePasswordReset,
		IgnoreUnknownUsernames: policy.IgnoreUnknownUsernames,
		AllowDomainDiscovery:   policy.AllowDomainDiscovery,
		DefaultRedirectURI:     policy.DefaultRedirectURI,

		PasswordCheckLifetime:      policy.PasswordCheckLifetime,
		ExternalLoginCheckLifetime: policy.ExternalLoginCheckLifetime,
		MFAInitSkipLifetime:        policy.MFAInitSkipLifetime,
		SecondFactorCheckLifetime:  policy.SecondFactorCheckLifetime,
		MultiFactorCheckLifetime:   policy.MultiFactorCheckLifetime,

		DisableLoginWithEmail: policy.DisableLoginWithEmail,
		DisableLoginWithPhone: policy.DisableLoginWithPhone,
	}
}

// DomainLoginPolicyToOrg converts a canonical domain.LoginPolicy back into an
// orgpolicy login policy.
func DomainLoginPolicyToOrg(policy *domain.LoginPolicy) orgpolicy.LoginPolicy {
	if policy == nil {
		return orgpolicy.DefaultLoginPolicy()
	}
	secondFactors := make([]orgpolicy.SecondFactorType, 0, len(policy.SecondFactors))
	for _, factor := range policy.SecondFactors {
		secondFactors = append(secondFactors, orgpolicy.SecondFactorType(factor))
	}
	multiFactors := make([]orgpolicy.MultiFactorType, 0, len(policy.MultiFactors))
	for _, factor := range policy.MultiFactors {
		multiFactors = append(multiFactors, orgpolicy.MultiFactorType(factor))
	}

	return orgpolicy.LoginPolicy{
		AllowUsernamePassword:  policy.AllowUsernamePassword,
		AllowRegister:          policy.AllowRegister,
		AllowExternalIDP:       policy.AllowExternalIDP,
		ForceMFA:               policy.ForceMFA,
		ForceMFALocalOnly:      policy.ForceMFALocalOnly,
		HidePasswordReset:      policy.HidePasswordReset,
		IgnoreUnknownUsernames: policy.IgnoreUnknownUsernames,
		AllowDomainDiscovery:   policy.AllowDomainDiscovery,
		DisableLoginWithEmail:  policy.DisableLoginWithEmail,
		DisableLoginWithPhone:  policy.DisableLoginWithPhone,
		PasswordlessType:       orgpolicy.PasswordlessType(policy.PasswordlessType),
		SecondFactors:          secondFactors,
		MultiFactors:           multiFactors,
		DefaultRedirectURI:     policy.DefaultRedirectURI,

		PasswordCheckLifetime:      policy.PasswordCheckLifetime,
		SecondFactorCheckLifetime:  policy.SecondFactorCheckLifetime,
		MultiFactorCheckLifetime:   policy.MultiFactorCheckLifetime,
		MFAInitSkipLifetime:        policy.MFAInitSkipLifetime,
		ExternalLoginCheckLifetime: policy.ExternalLoginCheckLifetime,
	}
}
