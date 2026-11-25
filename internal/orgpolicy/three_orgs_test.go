package orgpolicy_test

import (
	"context"
	"errors"
	"testing"

	"github.com/zitadel/zitadel/internal/orgpolicy"
)

// setupThreeOrgs builds the canonical acceptance scenario used across this
// file: one instance with exactly three organizations.
//
//   - org-acme:  fully customized brand and strict login policy
//   - org-beta:  only a partial brand/login change, everything else inherits
//   - org-gamma: no customization at all, fully inherited defaults
func setupThreeOrgs(t *testing.T) (*orgpolicy.Service, context.Context) {
	t.Helper()
	ctx := context.Background()
	svc, err := orgpolicy.NewService(orgpolicy.NewStore(), "inst-main")
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	for _, org := range []struct{ id, name string }{
		{"org-acme", "Acme Corp"},
		{"org-beta", "Beta Inc"},
		{"org-gamma", "Gamma LLC"},
	} {
		if _, err := svc.RegisterOrganization(ctx, org.id, org.name); err != nil {
			t.Fatalf("RegisterOrganization(%s): %v", org.id, err)
		}
	}

	acmeBrand := orgpolicy.Branding{
		Light: orgpolicy.ThemeColors{
			PrimaryColor:    "#ff5733",
			BackgroundColor: "#ffffff",
			WarnColor:       "#ffcc00",
			FontColor:       "#111111",
			LogoURL:         "/assets/acme/logo.svg",
			IconURL:         "/assets/acme/icon.svg",
		},
		Dark: orgpolicy.ThemeColors{
			PrimaryColor:    "#ff8a66",
			BackgroundColor: "#20242e",
			WarnColor:       "#ffd633",
			FontColor:       "#f5f5f5",
			LogoURL:         "/assets/acme/logo-dark.svg",
			IconURL:         "/assets/acme/icon-dark.svg",
		},
		FontURL:             "https://fonts.example.com/acme.woff2",
		HideLoginNameSuffix: true,
		ErrorMsgPopup:       true,
		DisableWatermark:    true,
		ThemeMode:           orgpolicy.ThemeLight,
	}
	if err := svc.UpdateBranding(ctx, "org-acme", acmeBrand); err != nil {
		t.Fatalf("UpdateBranding(acme): %v", err)
	}

	strictLogin := orgpolicy.DefaultLoginPolicy()
	strictLogin.AllowRegister = false
	strictLogin.ForceMFA = true
	strictLogin.ForceMFALocalOnly = true
	strictLogin.HidePasswordReset = true
	strictLogin.SecondFactors = []orgpolicy.SecondFactorType{
		orgpolicy.SecondFactorTOTP,
		orgpolicy.SecondFactorU2F,
	}
	if err := svc.UpdateLoginPolicy(ctx, "org-acme", strictLogin); err != nil {
		t.Fatalf("UpdateLoginPolicy(acme): %v", err)
	}

	// Beta only overrides one brand color, the theme mode and registration.
	if _, err := svc.PatchBranding(ctx, "org-beta", orgpolicy.BrandingPatch{
		Light:     &orgpolicy.ThemeColorsPatch{PrimaryColor: strPtr("#00aa88")},
		ThemeMode: themeModePtr(orgpolicy.ThemeDark),
	}); err != nil {
		t.Fatalf("PatchBranding(beta): %v", err)
	}
	if _, err := svc.PatchLoginPolicy(ctx, "org-beta", orgpolicy.LoginPolicyPatch{
		AllowRegister: boolPtr(false),
	}); err != nil {
		t.Fatalf("PatchLoginPolicy(beta): %v", err)
	}

	return svc, ctx
}

func strPtr(v string) *string                                 { return &v }
func boolPtr(v bool) *bool                                    { return &v }
func themeModePtr(v orgpolicy.ThemeMode) *orgpolicy.ThemeMode { return &v }

// TestThreeOrganizations_BrandingCorrectness verifies each of the three
// organizations resolves its own correct UI brand configuration.
func TestThreeOrganizations_BrandingCorrectness(t *testing.T) {
	svc, ctx := setupThreeOrgs(t)
	def := svc.GetDefaultBranding(ctx)

	sets, err := svc.ListOrganizationPolicySets(ctx)
	if err != nil {
		t.Fatalf("ListOrganizationPolicySets: %v", err)
	}
	if len(sets) != 3 {
		t.Fatalf("expected exactly 3 organizations, got %d", len(sets))
	}

	byID := make(map[string]*orgpolicy.OrganizationPolicySet, 3)
	for _, set := range sets {
		byID[set.Organization.ID] = set
		b := set.Branding.Value
		t.Logf("org=%s source=%s inherited=%v primaryLight=%s theme=%d logo=%s",
			set.Organization.ID, set.Branding.Source, set.Branding.Inherited,
			b.Light.PrimaryColor, b.ThemeMode, b.Light.LogoURL)
	}

	acme := byID["org-acme"].Branding
	if acme.Inherited || acme.Source != orgpolicy.SourceOrganization {
		t.Fatalf("acme branding must be organization owned, got inherited=%v source=%q", acme.Inherited, acme.Source)
	}
	if acme.Value.Light.PrimaryColor != "#ff5733" {
		t.Errorf("acme primary = %q, want #ff5733", acme.Value.Light.PrimaryColor)
	}
	if acme.Value.Light.LogoURL != "/assets/acme/logo.svg" {
		t.Errorf("acme logo = %q", acme.Value.Light.LogoURL)
	}
	if acme.Value.Dark.PrimaryColor != "#ff8a66" || acme.Value.Dark.LogoURL != "/assets/acme/logo-dark.svg" {
		t.Errorf("acme dark theme mismatch: %+v", acme.Value.Dark)
	}
	if !acme.Value.HideLoginNameSuffix || !acme.Value.ErrorMsgPopup || !acme.Value.DisableWatermark {
		t.Errorf("acme boolean brand flags not persisted: %+v", acme.Value)
	}
	if acme.Value.ThemeMode != orgpolicy.ThemeLight || acme.Value.FontURL == "" {
		t.Errorf("acme theme/font mismatch: mode=%d font=%q", acme.Value.ThemeMode, acme.Value.FontURL)
	}

	beta := byID["org-beta"].Branding
	if beta.Inherited || beta.Source != orgpolicy.SourceOrganization {
		t.Fatalf("beta branding patch should become an organization override")
	}
	if beta.Value.Light.PrimaryColor != "#00aa88" {
		t.Errorf("beta primary = %q, want #00aa88", beta.Value.Light.PrimaryColor)
	}
	// Unspecified fields must keep inheriting the instance default.
	if beta.Value.Light.BackgroundColor != def.Light.BackgroundColor {
		t.Errorf("beta background = %q, want inherited %q", beta.Value.Light.BackgroundColor, def.Light.BackgroundColor)
	}
	if beta.Value.Light.WarnColor != def.Light.WarnColor || beta.Value.Light.FontColor != def.Light.FontColor {
		t.Errorf("beta unoverridden light colors must inherit defaults: %+v vs %+v", beta.Value.Light, def.Light)
	}
	if beta.Value.Dark != def.Dark {
		t.Errorf("beta dark theme must be fully inherited: got %+v want %+v", beta.Value.Dark, def.Dark)
	}
	if beta.Value.ThemeMode != orgpolicy.ThemeDark {
		t.Errorf("beta theme mode = %d, want dark", beta.Value.ThemeMode)
	}
	if beta.Value.ErrorMsgPopup != def.ErrorMsgPopup || beta.Value.DisableWatermark != def.DisableWatermark {
		t.Errorf("beta unspecified boolean flags must inherit defaults")
	}

	gamma := byID["org-gamma"].Branding
	if !gamma.Inherited || gamma.Source != orgpolicy.SourceInstance {
		t.Fatalf("gamma branding must be inherited instance default, got inherited=%v source=%q", gamma.Inherited, gamma.Source)
	}
	if gamma.Value != def {
		t.Errorf("gamma brand must equal instance default\ngot  %+v\nwant %+v", gamma.Value, def)
	}
}

// TestThreeOrganizations_LoginPolicyInheritance verifies the login policy
// inheritance and override behavior for the three organizations.
func TestThreeOrganizations_LoginPolicyInheritance(t *testing.T) {
	svc, ctx := setupThreeOrgs(t)
	def := svc.GetDefaultLoginPolicy(ctx)

	acme, err := svc.GetLoginPolicy(ctx, "org-acme")
	if err != nil {
		t.Fatal(err)
	}
	if acme.Inherited {
		t.Fatal("acme login policy must be a custom organization policy")
	}
	if acme.Value.AllowRegister {
		t.Error("acme registration must be disabled")
	}
	if !acme.Value.ForceMFA || !acme.Value.ForceMFALocalOnly || !acme.Value.HidePasswordReset {
		t.Errorf("acme strict flags wrong: %+v", acme.Value)
	}
	if len(acme.Value.SecondFactors) != 2 ||
		acme.Value.SecondFactors[0] != orgpolicy.SecondFactorTOTP ||
		acme.Value.SecondFactors[1] != orgpolicy.SecondFactorU2F {
		t.Errorf("acme second factors wrong: %v", acme.Value.SecondFactors)
	}

	beta, err := svc.GetLoginPolicy(ctx, "org-beta")
	if err != nil {
		t.Fatal(err)
	}
	if beta.Value.AllowRegister {
		t.Error("beta registration patch must be applied")
	}
	// Everything except registration keeps inheriting the default.
	if beta.Value.AllowUsernamePassword != def.AllowUsernamePassword {
		t.Error("beta AllowUsernamePassword must inherit default")
	}
	if beta.Value.ForceMFA != def.ForceMFA {
		t.Error("beta ForceMFA must inherit default")
	}
	if beta.Value.PasswordlessType != def.PasswordlessType {
		t.Error("beta passwordless type must inherit default")
	}
	if len(beta.Value.SecondFactors) != len(def.SecondFactors) ||
		beta.Value.SecondFactors[0] != def.SecondFactors[0] {
		t.Errorf("beta factors must inherit defaults: %v", beta.Value.SecondFactors)
	}

	gamma, err := svc.GetLoginPolicy(ctx, "org-gamma")
	if err != nil {
		t.Fatal(err)
	}
	if !gamma.Inherited || gamma.Source != orgpolicy.SourceInstance {
		t.Fatalf("gamma login policy must inherit instance default")
	}
	if gamma.Value.AllowRegister != def.AllowRegister || gamma.Value.ForceMFA != def.ForceMFA {
		t.Error("gamma login policy must equal the instance default")
	}
}

// TestThreeOrganizations_PolicyIsolation is the key multi tenancy check:
// changing branding or policy of one organization can never alter the
// configuration resolved for the other two.
func TestThreeOrganizations_PolicyIsolation(t *testing.T) {
	svc, ctx := setupThreeOrgs(t)

	acmeBefore, _ := svc.GetBranding(ctx, "org-acme")
	gammaBefore, _ := svc.GetBranding(ctx, "org-gamma")

	// Mutate Acme heavily.
	acmeChanged := acmeBefore.Value
	acmeChanged.Light.PrimaryColor = "#000000"
	acmeChanged.ThemeMode = orgpolicy.ThemeDark
	if err := svc.UpdateBranding(ctx, "org-acme", acmeChanged); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.PatchLoginPolicy(ctx, "org-acme", orgpolicy.LoginPolicyPatch{
		AllowRegister: boolPtr(true),
		ForceMFA:      boolPtr(false),
	}); err != nil {
		t.Fatal(err)
	}

	// Mutate Beta as well.
	if _, err := svc.PatchBranding(ctx, "org-beta", orgpolicy.BrandingPatch{
		Light: &orgpolicy.ThemeColorsPatch{WarnColor: strPtr("#123456")},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.PatchLoginPolicy(ctx, "org-beta", orgpolicy.LoginPolicyPatch{
		ForceMFA: boolPtr(true),
	}); err != nil {
		t.Fatal(err)
	}

	// Gamma must be completely untouched and still inherited.
	gammaAfter, err := svc.GetBranding(ctx, "org-gamma")
	if err != nil {
		t.Fatal(err)
	}
	if !gammaAfter.Inherited || gammaAfter.Value != gammaBefore.Value {
		t.Fatalf("gamma branding changed due to another org mutation: before=%+v after=%+v", gammaBefore.Value, gammaAfter.Value)
	}
	gammaLogin, _ := svc.GetLoginPolicy(ctx, "org-gamma")
	if !gammaLogin.Inherited || gammaLogin.Value.ForceMFA != svc.GetDefaultLoginPolicy(ctx).ForceMFA {
		t.Fatal("gamma login policy must remain the inherited default")
	}

	// Beta must only show its own two independent changes; the Acme mutation
	// must not leak into it.
	betaAfter, err := svc.GetBranding(ctx, "org-beta")
	if err != nil {
		t.Fatal(err)
	}
	if betaAfter.Value.Light.PrimaryColor != "#00aa88" {
		t.Errorf("beta primary leaked/acme-contaminated: %q", betaAfter.Value.Light.PrimaryColor)
	}
	if betaAfter.Value.Light.WarnColor != "#123456" {
		t.Errorf("beta warn patch missing: %q", betaAfter.Value.Light.WarnColor)
	}
	if betaAfter.Value.Light.PrimaryColor == "#000000" {
		t.Error("acme primary color leaked into beta")
	}
	betaLogin, _ := svc.GetLoginPolicy(ctx, "org-beta")
	if !betaLogin.Value.ForceMFA {
		t.Error("beta ForceMFA patch missing")
	}
	if betaLogin.Value.AllowRegister {
		t.Error("beta AllowRegister should remain false independently of acme")
	}

	// Acme must reflect exactly its own changes.
	acmeAfter, err := svc.GetBranding(ctx, "org-acme")
	if err != nil {
		t.Fatal(err)
	}
	if acmeAfter.Value.Light.PrimaryColor != "#000000" || acmeAfter.Value.ThemeMode != orgpolicy.ThemeDark {
		t.Errorf("acme own mutation not stored: %+v", acmeAfter.Value)
	}
	acmeLogin, _ := svc.GetLoginPolicy(ctx, "org-acme")
	if !acmeLogin.Value.AllowRegister || acmeLogin.Value.ForceMFA {
		t.Errorf("acme login mutation not stored: register=%v mfa=%v", acmeLogin.Value.AllowRegister, acmeLogin.Value.ForceMFA)
	}

	// Resetting Acme removes only Acme's overrides; Beta and Gamma stay.
	if err := svc.ResetBranding(ctx, "org-acme"); err != nil {
		t.Fatal(err)
	}
	if err := svc.ResetLoginPolicy(ctx, "org-acme"); err != nil {
		t.Fatal(err)
	}
	acmeResetBrand, _ := svc.GetBranding(ctx, "org-acme")
	if !acmeResetBrand.Inherited || acmeResetBrand.Value != svc.GetDefaultBranding(ctx) {
		t.Error("acme branding should inherit defaults after reset")
	}
	acmeResetLogin, _ := svc.GetLoginPolicy(ctx, "org-acme")
	if !acmeResetLogin.Inherited {
		t.Error("acme login policy should inherit defaults after reset")
	}
	betaStill, _ := svc.GetBranding(ctx, "org-beta")
	if betaStill.Value.Light.WarnColor != "#123456" {
		t.Error("resetting acme altered beta")
	}

	// Unknown organizations are rejected instead of silently sharing data.
	_, err = svc.GetBranding(ctx, "org-does-not-exist")
	if !errors.Is(err, orgpolicy.ErrOrganizationNotFound) {
		t.Fatalf("expected ErrOrganizationNotFound, got %v", err)
	}
}
