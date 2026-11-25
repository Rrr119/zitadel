package iamcompat_test

import (
	"context"
	"testing"

	"github.com/zitadel/zitadel/internal/domain"
	"github.com/zitadel/zitadel/internal/orgpolicy"
	"github.com/zitadel/zitadel/internal/orgpolicy/iamcompat"
)

// TestThreeOrgs_MapToCanonicalIAMTypes runs the same three organization
// scenario through the compatibility mapping and asserts the canonical
// ZITADEL policy structs carry the correct ownership and Default flag.
func TestThreeOrgs_MapToCanonicalIAMTypes(t *testing.T) {
	ctx := context.Background()
	store := orgpolicy.NewStore()
	svc, err := orgpolicy.NewService(store, "inst-1")
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"org-acme", "org-beta", "org-gamma"} {
		if _, err := svc.RegisterOrganization(ctx, id, id); err != nil {
			t.Fatal(err)
		}
	}

	custom := orgpolicy.DefaultBranding()
	custom.Light.PrimaryColor = "#ff5733"
	if err := svc.UpdateBranding(ctx, "org-acme", custom); err != nil {
		t.Fatal(err)
	}
	strict := orgpolicy.DefaultLoginPolicy()
	strict.AllowRegister = false
	strict.ForceMFA = true
	if err := svc.UpdateLoginPolicy(ctx, "org-acme", strict); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.PatchBranding(ctx, "org-beta", orgpolicy.BrandingPatch{
		Light: &orgpolicy.ThemeColorsPatch{PrimaryColor: strPtr("#00aa88")},
	}); err != nil {
		t.Fatal(err)
	}

	sets, err := svc.ListOrganizationPolicySets(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(sets) != 3 {
		t.Fatalf("want 3 org sets, got %d", len(sets))
	}

	for _, set := range sets {
		orgID := set.Organization.ID

		label := iamcompat.BrandingToLabelPolicy(set.Branding.Value, "inst-1", orgID, set.Branding.Inherited)
		if label.Default != set.Branding.Inherited {
			t.Errorf("%s label Default=%v want %v", orgID, label.Default, set.Branding.Inherited)
		}
		if label.ResourceOwner != orgID || label.InstanceID != "inst-1" {
			t.Errorf("%s label ownership wrong: %+v", orgID, label.ObjectRoot)
		}
		if err := label.IsValid(); err != nil {
			t.Errorf("%s mapped label policy must be valid in domain: %v", orgID, err)
		}

		login := iamcompat.LoginPolicyToDomain(set.LoginPolicy.Value, "inst-1", orgID, set.LoginPolicy.Inherited)
		if login.Default != set.LoginPolicy.Inherited {
			t.Errorf("%s login Default=%v want %v", orgID, login.Default, set.LoginPolicy.Inherited)
		}
		if login.ResourceOwner != orgID {
			t.Errorf("%s login ResourceOwner=%q", orgID, login.ResourceOwner)
		}
	}

	// Spot check Acme values survived mapping.
	acme, err := svc.GetBranding(ctx, "org-acme")
	if err != nil {
		t.Fatal(err)
	}
	label := iamcompat.BrandingToLabelPolicy(acme.Value, "inst-1", "org-acme", false)
	if label.PrimaryColor != "#ff5733" || label.Default {
		t.Fatalf("acme label mapping wrong: primary=%s default=%v", label.PrimaryColor, label.Default)
	}

	// Gamma is inherited and therefore maps to a canonical default policy.
	gamma, _ := svc.GetBranding(ctx, "org-gamma")
	gammaLabel := iamcompat.BrandingToLabelPolicy(gamma.Value, "inst-1", "org-gamma", gamma.Inherited)
	if !gammaLabel.Default {
		t.Fatal("gamma label must be flagged Default like an instance default response")
	}
}

func TestBrandingRoundTripThroughDomain(t *testing.T) {
	original := orgpolicy.DefaultBranding()
	original.Light.PrimaryColor = "#123456"
	original.Dark.LogoURL = "/assets/dark-logo.svg"
	original.ErrorMsgPopup = true
	original.ThemeMode = orgpolicy.ThemeDark

	label := iamcompat.BrandingToLabelPolicy(original, "inst", "org", false)
	roundTripped := iamcompat.LabelPolicyToBranding(label)

	if roundTripped != original {
		t.Fatalf("branding round trip mismatch:\ngot  %+v\nwant %+v", roundTripped, original)
	}
}

func TestLoginPolicyRoundTripThroughDomain(t *testing.T) {
	original := orgpolicy.DefaultLoginPolicy()
	original.AllowRegister = false
	original.ForceMFA = true
	original.SecondFactors = []orgpolicy.SecondFactorType{
		orgpolicy.SecondFactorTOTP,
		orgpolicy.SecondFactorOTPSMS,
	}

	mapped := iamcompat.LoginPolicyToDomain(original, "inst", "org", false)
	if mapped.PasswordlessType != domain.PasswordlessTypeAllowed {
		t.Fatalf("passwordless enum mapping wrong: %v", mapped.PasswordlessType)
	}
	roundTripped := iamcompat.DomainLoginPolicyToOrg(mapped)

	if roundTripped.AllowRegister != original.AllowRegister ||
		roundTripped.ForceMFA != original.ForceMFA ||
		len(roundTripped.SecondFactors) != len(original.SecondFactors) {
		t.Fatalf("login policy round trip mismatch: %+v", roundTripped)
	}
	for i := range original.SecondFactors {
		if roundTripped.SecondFactors[i] != original.SecondFactors[i] {
			t.Fatalf("factor %d mismatch: %v vs %v", i, roundTripped.SecondFactors, original.SecondFactors)
		}
	}
}

func strPtr(v string) *string { return &v }
