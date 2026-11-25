package orgpolicy

import (
	"net/url"
	"regexp"
)

// colorPattern mirrors the validation used by the canonical ZITADEL label
// policy (internal/domain/policy_label.go): empty or #rgb / #rrggbb.
var colorPattern = regexp.MustCompile(`^$|^#(?:[A-Fa-f0-9]{6}|[A-Fa-f0-9]{3})$`)

// ThemeMode controls which color theme the hosted UI renders. The numeric
// values intentionally match domain.LabelPolicyThemeMode and the v3
// BrandingPolicyThemeMode enumerations.
type ThemeMode int32

const (
	ThemeAuto ThemeMode = iota
	ThemeLight
	ThemeDark
)

// ThemeColors groups the customizable colors and asset URLs of one theme.
type ThemeColors struct {
	PrimaryColor    string `json:"primaryColor,omitempty"`
	BackgroundColor string `json:"backgroundColor,omitempty"`
	WarnColor       string `json:"warnColor,omitempty"`
	FontColor       string `json:"fontColor,omitempty"`
	LogoURL         string `json:"logoUrl,omitempty"`
	IconURL         string `json:"iconUrl,omitempty"`
}

// ThemeColorsPatch is the optional-field counterpart of ThemeColors. A nil
// pointer means "keep the current value"; a non-nil pointer replaces it.
type ThemeColorsPatch struct {
	PrimaryColor    *string `json:"primaryColor,omitempty"`
	BackgroundColor *string `json:"backgroundColor,omitempty"`
	WarnColor       *string `json:"warnColor,omitempty"`
	FontColor       *string `json:"fontColor,omitempty"`
	LogoURL         *string `json:"logoUrl,omitempty"`
	IconURL         *string `json:"iconUrl,omitempty"`
}

// Branding is the resolved UI brand / label policy of an organization or the
// instance. It uses value fields only so a Branding can be copied without
// aliasing nested data.
type Branding struct {
	Light ThemeColors `json:"light"`
	Dark  ThemeColors `json:"dark"`

	FontURL string `json:"fontUrl,omitempty"`

	HideLoginNameSuffix bool `json:"hideLoginNameSuffix,omitempty"`
	ErrorMsgPopup       bool `json:"errorMsgPopup,omitempty"`
	DisableWatermark    bool `json:"disableWatermark,omitempty"`

	ThemeMode ThemeMode `json:"themeMode"`
}

// BrandingPatch carries optional organization overrides. Nil fields inherit
// the value from the currently effective (typically instance default) brand.
type BrandingPatch struct {
	Light *ThemeColorsPatch `json:"light,omitempty"`
	Dark  *ThemeColorsPatch `json:"dark,omitempty"`

	FontURL *string `json:"fontUrl,omitempty"`

	HideLoginNameSuffix *bool `json:"hideLoginNameSuffix,omitempty"`
	ErrorMsgPopup       *bool `json:"errorMsgPopup,omitempty"`
	DisableWatermark    *bool `json:"disableWatermark,omitempty"`

	ThemeMode *ThemeMode `json:"themeMode,omitempty"`
}

// DefaultBranding returns the built-in instance brand used when no explicit
// default was configured.
func DefaultBranding() Branding {
	light := ThemeColors{
		PrimaryColor:    "#5469d4",
		BackgroundColor: "#fafafa",
		WarnColor:       "#cd3d56",
		FontColor:       "#000000",
	}
	dark := ThemeColors{
		PrimaryColor:    "#6f82e6",
		BackgroundColor: "#1b1f2a",
		WarnColor:       "#ff6b81",
		FontColor:       "#ffffff",
	}
	return Branding{
		Light:               light,
		Dark:                dark,
		HideLoginNameSuffix: false,
		ErrorMsgPopup:       false,
		DisableWatermark:    false,
		ThemeMode:           ThemeAuto,
	}
}

// Clone returns a copy of the brand. Branding currently has only value
// fields, so a direct copy is already isolated.
func (b Branding) Clone() Branding {
	return b
}

// Apply returns a copy of b with every non-nil patch field applied. The
// receiver is not modified, which keeps resolution side effect free.
func (b Branding) Apply(patch BrandingPatch) Branding {
	out := b
	if patch.Light != nil {
		out.Light = applyThemeColorsPatch(out.Light, *patch.Light)
	}
	if patch.Dark != nil {
		out.Dark = applyThemeColorsPatch(out.Dark, *patch.Dark)
	}
	if patch.FontURL != nil {
		out.FontURL = *patch.FontURL
	}
	if patch.HideLoginNameSuffix != nil {
		out.HideLoginNameSuffix = *patch.HideLoginNameSuffix
	}
	if patch.ErrorMsgPopup != nil {
		out.ErrorMsgPopup = *patch.ErrorMsgPopup
	}
	if patch.DisableWatermark != nil {
		out.DisableWatermark = *patch.DisableWatermark
	}
	if patch.ThemeMode != nil {
		out.ThemeMode = *patch.ThemeMode
	}
	return out
}

func applyThemeColorsPatch(current ThemeColors, patch ThemeColorsPatch) ThemeColors {
	out := current
	if patch.PrimaryColor != nil {
		out.PrimaryColor = *patch.PrimaryColor
	}
	if patch.BackgroundColor != nil {
		out.BackgroundColor = *patch.BackgroundColor
	}
	if patch.WarnColor != nil {
		out.WarnColor = *patch.WarnColor
	}
	if patch.FontColor != nil {
		out.FontColor = *patch.FontColor
	}
	if patch.LogoURL != nil {
		out.LogoURL = *patch.LogoURL
	}
	if patch.IconURL != nil {
		out.IconURL = *patch.IconURL
	}
	return out
}

// Validate checks colors, theme mode and URL fields and returns a
// ValidationError collecting every problem.
func (b Branding) Validate() error {
	var issues []FieldError

	validateThemeColors := func(prefix string, colors ThemeColors) {
		colorFields := []struct {
			name  string
			value string
		}{
			{"primaryColor", colors.PrimaryColor},
			{"backgroundColor", colors.BackgroundColor},
			{"warnColor", colors.WarnColor},
			{"fontColor", colors.FontColor},
		}
		for _, field := range colorFields {
			if !colorPattern.MatchString(field.value) {
				issues = append(issues, FieldError{
					Field:   prefix + "." + field.name,
					Message: "must be empty or a #rgb/#rrggbb hex color, got " + field.value,
				})
			}
		}
		for _, field := range []struct {
			name  string
			value string
		}{
			{"logoUrl", colors.LogoURL},
			{"iconUrl", colors.IconURL},
		} {
			if err := validateAssetURL(field.value); err != nil {
				issues = append(issues, FieldError{Field: prefix + "." + field.name, Message: err.Error()})
			}
		}
	}

	validateThemeColors("light", b.Light)
	validateThemeColors("dark", b.Dark)

	if err := validateAssetURL(b.FontURL); err != nil {
		issues = append(issues, FieldError{Field: "fontUrl", Message: err.Error()})
	}
	if b.ThemeMode < ThemeAuto || b.ThemeMode > ThemeDark {
		issues = append(issues, FieldError{Field: "themeMode", Message: "must be one of auto(0), light(1), dark(2)"})
	}
	return validationError(issues)
}

// validateAssetURL accepts empty values (asset not set), root relative
// paths served by ZITADEL and absolute http/https URLs.
func validateAssetURL(raw string) error {
	if raw == "" {
		return nil
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return errInvalidURL
	}
	if parsed.IsAbs() {
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return errInvalidURLScheme
		}
		if parsed.Host == "" {
			return errInvalidURLHost
		}
		return nil
	}
	if raw[0] != '/' {
		return errInvalidURLRelative
	}
	return nil
}

type urlValidationError string

func (e urlValidationError) Error() string { return string(e) }

const (
	errInvalidURL         = urlValidationError("must be a valid URL")
	errInvalidURLScheme   = urlValidationError("absolute URL must use http or https")
	errInvalidURLHost     = urlValidationError("absolute URL must include a host")
	errInvalidURLRelative = urlValidationError("relative URL must start with /")
)
