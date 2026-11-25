package orgpolicy_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/zitadel/zitadel/internal/orgpolicy"
)

func TestNewService_RequiresInstanceID(t *testing.T) {
	if _, err := orgpolicy.NewService(orgpolicy.NewStore(), ""); !errors.Is(err, orgpolicy.ErrInvalidInstanceID) {
		t.Fatalf("expected ErrInvalidInstanceID, got %v", err)
	}
}

func TestStore_UnknownInstanceReadsAreSafe(t *testing.T) {
	store := orgpolicy.NewStore()

	if orgs := store.ListOrganizations("ghost"); len(orgs) != 0 {
		t.Fatalf("unknown instance must have no organizations, got %d", len(orgs))
	}
	if _, err := store.FindOrganization("ghost", "org-1"); !errors.Is(err, orgpolicy.ErrOrganizationNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
	if _, _, err := store.GetOrgBranding("ghost", "org-1"); !errors.Is(err, orgpolicy.ErrOrganizationNotFound) {
		t.Fatalf("expected not found for branding, got %v", err)
	}
	// Defaults are still served for an uninitialized instance.
	if store.GetDefaultBranding("ghost").Light.PrimaryColor == "" {
		t.Fatal("expected built-in default branding for unknown instance")
	}
	if !store.GetDefaultLoginPolicy("ghost").AllowUsernamePassword {
		t.Fatal("expected built-in default login policy for unknown instance")
	}
}

func TestRegisterOrganization_ValidatesAndDeduplicates(t *testing.T) {
	ctx := context.Background()
	svc, _ := orgpolicy.NewService(orgpolicy.NewStore(), "inst")

	if _, err := svc.RegisterOrganization(ctx, "", "No ID"); !errors.Is(err, orgpolicy.ErrOrganizationNotFound) {
		t.Fatalf("empty org id should be rejected, got %v", err)
	}
	if _, err := svc.RegisterOrganization(ctx, "org-1", "One"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.RegisterOrganization(ctx, "org-1", "One duplicate"); !errors.Is(err, orgpolicy.ErrOrganizationExists) {
		t.Fatalf("expected ErrOrganizationExists, got %v", err)
	}

	ids := svc.ListOrganizations(ctx)
	if len(ids) != 1 || ids[0].Name != "One" || ids[0].InstanceID != "inst" {
		t.Fatalf("unexpected org list: %+v", ids)
	}
}

func TestInstances_AreIsolated(t *testing.T) {
	ctx := context.Background()
	store := orgpolicy.NewStore()

	one, err := orgpolicy.NewService(store, "inst-one")
	if err != nil {
		t.Fatal(err)
	}
	two, err := orgpolicy.NewService(store, "inst-two")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := one.RegisterOrganization(ctx, "shared-org-id", "In One"); err != nil {
		t.Fatal(err)
	}
	if _, err := two.RegisterOrganization(ctx, "shared-org-id", "In Two"); err != nil {
		t.Fatal(err)
	}

	brandOne := orgpolicy.DefaultBranding()
	brandOne.Light.PrimaryColor = "#111111"
	if err := one.UpdateBranding(ctx, "shared-org-id", brandOne); err != nil {
		t.Fatal(err)
	}

	// The same organization id on instance two must not see instance one's
	// customization and must resolve its own inherited default.
	gotTwo, err := two.GetBranding(ctx, "shared-org-id")
	if err != nil {
		t.Fatal(err)
	}
	if !gotTwo.Inherited || gotTwo.Value.Light.PrimaryColor != two.GetDefaultBranding(ctx).Light.PrimaryColor {
		t.Fatalf("instance leakage: %+v", gotTwo)
	}

	// Changing instance two's default must not affect instance one or its orgs.
	defaultTwo := two.GetDefaultBranding(ctx)
	defaultTwo.Light.PrimaryColor = "#222222"
	if err := two.UpdateDefaultBranding(ctx, defaultTwo); err != nil {
		t.Fatal(err)
	}
	gotOne, _ := one.GetBranding(ctx, "shared-org-id")
	if gotOne.Value.Light.PrimaryColor != "#111111" {
		t.Fatalf("instance two default change leaked into instance one: %q", gotOne.Value.Light.PrimaryColor)
	}

	// An org that exists only on instance one is not addressable on two.
	if _, err := two.GetBranding(ctx, "org-acme"); !errors.Is(err, orgpolicy.ErrOrganizationNotFound) {
		t.Fatalf("cross-instance org should be invisible, got %v", err)
	}
}

func TestAuditEvents_RecordPolicyLifecycle(t *testing.T) {
	ctx := context.Background()
	store := orgpolicy.NewStore()
	svc, _ := orgpolicy.NewService(store, "inst-audit")
	if _, err := svc.RegisterOrganization(ctx, "org-1", "One"); err != nil {
		t.Fatal(err)
	}
	if err := svc.UpdateBranding(ctx, "org-1", orgpolicy.DefaultBranding()); err != nil {
		t.Fatal(err)
	}
	if err := svc.ResetBranding(ctx, "org-1"); err != nil {
		t.Fatal(err)
	}

	events := store.AuditEvents("inst-audit")
	if len(events) != 3 {
		t.Fatalf("expected 3 audit events, got %d: %+v", len(events), events)
	}
	if events[0].Action != orgpolicy.ActionRegistered || events[1].Scope != orgpolicy.ScopeBranding ||
		events[2].Action != orgpolicy.ActionResetOrg {
		t.Fatalf("unexpected audit trail: %+v", events)
	}
	if events[0].At.IsZero() {
		t.Fatal("audit event timestamp must be populated")
	}
}

func TestService_ConcurrentOrgMutationsStayIsolated(t *testing.T) {
	ctx := context.Background()
	svc, _ := orgpolicy.NewService(orgpolicy.NewStore(), "inst-race")
	const orgs = 8
	for i := 0; i < orgs; i++ {
		if _, err := svc.RegisterOrganization(ctx, fmt.Sprintf("org-%d", i), fmt.Sprintf("Org %d", i)); err != nil {
			t.Fatal(err)
		}
	}

	var wg sync.WaitGroup
	for i := 0; i < orgs; i++ {
		orgID := fmt.Sprintf("org-%d", i)
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			color := fmt.Sprintf("#%06x", (i*0x10101)%0xffffff)
			for j := 0; j < 25; j++ {
				_, err := svc.PatchBranding(ctx, orgID, orgpolicy.BrandingPatch{
					Light: &orgpolicy.ThemeColorsPatch{PrimaryColor: strPtr(color)},
				})
				if err != nil {
					t.Errorf("patch %s: %v", orgID, err)
					return
				}
				got, err := svc.GetBranding(ctx, orgID)
				if err != nil {
					t.Errorf("get %s: %v", orgID, err)
					return
				}
				if got.Value.Light.PrimaryColor != color {
					t.Errorf("%s observed foreign color %q want %q", orgID, got.Value.Light.PrimaryColor, color)
					return
				}
			}
		}(i)
	}
	wg.Wait()
}
