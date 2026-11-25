package orgpolicy_test

import (
	"context"
	"fmt"

	"github.com/zitadel/zitadel/internal/orgpolicy"
)

// Example_threeOrganizations shows the core acceptance scenario: three
// organizations on one instance keep isolated branding and login policies,
// with unset organization settings inherited from the instance default.
func Example_threeOrganizations() {
	ctx := context.Background()
	svc, _ := orgpolicy.NewService(orgpolicy.NewStore(), "inst-demo")

	for _, id := range []string{"org-acme", "org-beta", "org-gamma"} {
		_, _ = svc.RegisterOrganization(ctx, id, id)
	}

	acmeBrand := orgpolicy.DefaultBranding()
	acmeBrand.Light.PrimaryColor = "#ff5733"
	_ = svc.UpdateBranding(ctx, "org-acme", acmeBrand)
	acmeLogin := orgpolicy.DefaultLoginPolicy()
	acmeLogin.AllowRegister = false
	acmeLogin.ForceMFA = true
	_ = svc.UpdateLoginPolicy(ctx, "org-acme", acmeLogin)

	_, _ = svc.PatchBranding(ctx, "org-beta", orgpolicy.BrandingPatch{
		Light: &orgpolicy.ThemeColorsPatch{PrimaryColor: strPtr("#00aa88")},
	})
	_, _ = svc.PatchLoginPolicy(ctx, "org-beta", orgpolicy.LoginPolicyPatch{
		AllowRegister: boolPtr(false),
	})

	sets, _ := svc.ListOrganizationPolicySets(ctx)
	for _, set := range sets {
		fmt.Printf("%s: source=%s primary=%s register=%t mfa=%t\n",
			set.Organization.ID,
			set.Branding.Source,
			set.Branding.Value.Light.PrimaryColor,
			set.LoginPolicy.Value.AllowRegister,
			set.LoginPolicy.Value.ForceMFA,
		)
	}
	// Output:
	// org-acme: source=organization primary=#ff5733 register=false mfa=true
	// org-beta: source=organization primary=#00aa88 register=false mfa=false
	// org-gamma: source=instance_default primary=#5469d4 register=true mfa=false
}
