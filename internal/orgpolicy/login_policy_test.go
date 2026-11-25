package orgpolicy_test

import (
	"testing"
	"time"

	"github.com/zitadel/zitadel/internal/orgpolicy"
)

func passlessPtr(v orgpolicy.PasswordlessType) *orgpolicy.PasswordlessType { return &v }

func TestLoginPolicy_DefaultIsValid(t *testing.T) {
	if err := orgpolicy.DefaultLoginPolicy().Validate(); err != nil {
		t.Fatalf("default login policy must be valid: %v", err)
	}
}

func TestLoginPolicy_ValidateRedirectURIAndLifetime(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*orgpolicy.LoginPolicy)
		wantErr bool
	}{
		{"empty redirect allowed", func(p *orgpolicy.LoginPolicy) { p.DefaultRedirectURI = "" }, false},
		{"https redirect", func(p *orgpolicy.LoginPolicy) { p.DefaultRedirectURI = "https://app.example/callback" }, false},
		{"custom scheme", func(p *orgpolicy.LoginPolicy) { p.DefaultRedirectURI = "myapp://callback" }, false},
		{"relative redirect rejected", func(p *orgpolicy.LoginPolicy) { p.DefaultRedirectURI = "/relative" }, true},
		{"negative lifetime rejected", func(p *orgpolicy.LoginPolicy) { p.ForceMFA = false; p.PasswordCheckLifetime = -time.Hour }, true},
		{"bad passwordless enum", func(p *orgpolicy.LoginPolicy) { p.PasswordlessType = 99 }, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := orgpolicy.DefaultLoginPolicy()
			tc.mutate(&p)
			err := p.Validate()
			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestLoginPolicy_ApplyPatchAndCloneIsolation(t *testing.T) {
	original := orgpolicy.DefaultLoginPolicy()
	patched := original.Apply(orgpolicy.LoginPolicyPatch{
		AllowRegister:    boolPtr(false),
		ForceMFA:         boolPtr(true),
		PasswordlessType: passlessPtr(orgpolicy.PasswordlessNotAllowed),
		SecondFactors:    []orgpolicy.SecondFactorType{orgpolicy.SecondFactorU2F},
	})

	if patched.AllowRegister || !patched.ForceMFA {
		t.Fatalf("bool patch wrong: %+v", patched)
	}
	if patched.PasswordlessType != orgpolicy.PasswordlessNotAllowed {
		t.Fatal("passwordless patch not applied")
	}
	if len(patched.SecondFactors) != 1 || patched.SecondFactors[0] != orgpolicy.SecondFactorU2F {
		t.Fatalf("factor replacement wrong: %v", patched.SecondFactors)
	}
	// Replacing the slice on the patch target must not alias the original.
	patched.SecondFactors[0] = orgpolicy.SecondFactorOTPSMS
	if original.SecondFactors[0] != orgpolicy.SecondFactorTOTP {
		t.Fatal("clone aliases second factor backing array")
	}

	// Nil factor slices preserve existing factors.
	preserved := original.Apply(orgpolicy.LoginPolicyPatch{HidePasswordReset: boolPtr(true)})
	if !preserved.HidePasswordReset || len(preserved.SecondFactors) != len(original.SecondFactors) {
		t.Fatal("nil factor patch should preserve factors and apply bool")
	}
}

func TestLoginPolicy_StoredCloneProtectsState(t *testing.T) {
	svc, ctx := setupThreeOrgs(t)

	policy := orgpolicy.DefaultLoginPolicy()
	if err := svc.UpdateLoginPolicy(ctx, "org-gamma", policy); err != nil {
		t.Fatal(err)
	}
	// Mutate the caller's slice after storing; stored state must be unaffected.
	policy.SecondFactors[0] = orgpolicy.SecondFactorOTPEmail

	got, err := svc.GetLoginPolicy(ctx, "org-gamma")
	if err != nil {
		t.Fatal(err)
	}
	if got.Value.SecondFactors[0] != orgpolicy.SecondFactorTOTP {
		t.Fatalf("store aliased caller slice: %v", got.Value.SecondFactors)
	}

	// Mutating a returned policy must also not affect subsequent reads.
	got.Value.SecondFactors[0] = orgpolicy.SecondFactorOTPSMS
	again, _ := svc.GetLoginPolicy(ctx, "org-gamma")
	if again.Value.SecondFactors[0] != orgpolicy.SecondFactorTOTP {
		t.Fatal("returned policy aliases stored state")
	}
}
