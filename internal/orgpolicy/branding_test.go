package orgpolicy_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/zitadel/zitadel/internal/orgpolicy"
)

func TestBranding_ValidateAcceptsCanonicalValues(t *testing.T) {
	b := orgpolicy.DefaultBranding()
	b.Light.LogoURL = "/assets/logo.svg"
	b.Light.IconURL = "https://cdn.example.com/icon.png"
	b.FontURL = "https://fonts.example.com/font.woff2"
	if err := b.Validate(); err != nil {
		t.Fatalf("expected default brand to be valid: %v", err)
	}
}

func TestBranding_ValidateRejectsBadColorsAndURLs(t *testing.T) {
	b := orgpolicy.DefaultBranding()
	b.Light.PrimaryColor = "blue"
	b.Dark.WarnColor = "#12"
	b.Light.LogoURL = "ftp://example.com/logo.svg"
	b.FontURL = "relative.woff2"

	err := b.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !errors.Is(err, orgpolicy.ErrInvalidPolicy) {
		t.Fatalf("expected ErrInvalidPolicy wrap, got %v", err)
	}
	message := err.Error()
	for _, want := range []string{"light.primaryColor", "dark.warnColor", "light.logoUrl", "fontUrl"} {
		if !strings.Contains(message, want) {
			t.Errorf("validation error %q missing field %q", message, want)
		}
	}
}

func TestBranding_ApplyPatchDoesNotMutateReceiver(t *testing.T) {
	original := orgpolicy.DefaultBranding()
	patched := original.Apply(orgpolicy.BrandingPatch{
		Light:               &orgpolicy.ThemeColorsPatch{PrimaryColor: strPtr("#abcdef")},
		ErrorMsgPopup:       boolPtr(true),
		HideLoginNameSuffix: boolPtr(true),
		ThemeMode:           themeModePtr(orgpolicy.ThemeDark),
	})

	if patched.Light.PrimaryColor != "#abcdef" {
		t.Fatalf("patch color not applied: %q", patched.Light.PrimaryColor)
	}
	if original.Light.PrimaryColor == "#abcdef" {
		t.Fatal("Apply mutated the receiver")
	}
	if !patched.ErrorMsgPopup || !patched.HideLoginNameSuffix || patched.ThemeMode != orgpolicy.ThemeDark {
		t.Fatalf("scalar patch not applied: %+v", patched)
	}
	// Unspecified nested fields remain inherited.
	if patched.Light.BackgroundColor != original.Light.BackgroundColor {
		t.Fatal("unspecified color should be preserved")
	}
}

func TestBranding_JSONRoundTrip(t *testing.T) {
	b := orgpolicy.DefaultBranding()
	b.Light.PrimaryColor = "#123123"
	b.DisableWatermark = true

	data, err := json.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}
	var decoded orgpolicy.Branding
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Light.PrimaryColor != b.Light.PrimaryColor || decoded.DisableWatermark != b.DisableWatermark {
		t.Fatalf("json round trip mismatch: %s", string(data))
	}
}
