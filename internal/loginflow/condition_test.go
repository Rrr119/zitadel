package loginflow

import (
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zitadel/zitadel/internal/domain"
)

type testAction struct {
	id string
}

func (a testAction) GetID() string {
	return a.id
}

type failingProvider struct {
	err error
}

func (p failingProvider) ConditionForAction(_, _ string) (*Condition, error) {
	return nil, p.err
}

// TestFlow_ConditionOrchestration holds the 10 organization scoped login flow
// scenarios: legacy compatibility, every condition field, the AND/OR semantics
// and successful vs. failed authentication.
func TestFlow_ConditionOrchestration(t *testing.T) {
	t.Parallel()

	authenticated := FlowContext{
		OrgID:                "org-1",
		RequestedOrgID:       "org-1",
		UserOrgID:            "org-2",
		AuthMethod:           "password",
		IDPID:                "idp-1",
		UserAgent:            "Mozilla/5.0 (iPhone)",
		RemoteIP:             net.ParseIP("10.0.0.5"),
		Authenticated:        true,
		AuthenticationFailed: false,
	}

	tests := []struct {
		name      string
		condition *Condition
		ctx       *FlowContext
		want      bool
	}{
		{
			name:      "1. nil condition matches every flow (legacy compatibility)",
			condition: nil,
			ctx:       &authenticated,
			want:      true,
		},
		{
			name:      "2. empty condition matches every flow",
			condition: &Condition{},
			ctx:       &authenticated,
			want:      true,
		},
		{
			name: "3. org condition matches requested organization",
			condition: &Condition{
				Organizations: []string{"org-1"},
			},
			ctx:  &authenticated,
			want: true,
		},
		{
			name: "4. org condition matches among multiple allowed organizations",
			condition: &Condition{
				Organizations: []string{"org-9", "org-2"},
			},
			ctx: &FlowContext{
				OrgID:          "org-2",
				RequestedOrgID: "org-2",
				UserOrgID:      "org-2",
				AuthMethod:     "password",
				UserAgent:      authenticated.UserAgent,
				RemoteIP:       authenticated.RemoteIP,
			},
			want: true,
		},
		{
			name: "5. org condition rejects unknown organization",
			condition: &Condition{
				Organizations: []string{"org-other"},
			},
			ctx:  &authenticated,
			want: false,
		},
		{
			name: "6. auth method condition matches password flow",
			condition: &Condition{
				AuthMethods: []string{"OTP", "password"},
			},
			ctx:  &authenticated,
			want: true,
		},
		{
			name: "7. IDP and user agent regex conditions both match (AND)",
			condition: &Condition{
				IDPIDs:           []string{"idp-1"},
				UserAgentRegexes: []string{`iPhone`, `Android`},
			},
			ctx:  &authenticated,
			want: true,
		},
		{
			name: "8. AND condition fails when one field does not match",
			condition: &Condition{
				Organizations:    []string{"org-1"},
				UserAgentRegexes: []string{`Windows`},
			},
			ctx:  &authenticated,
			want: false,
		},
		{
			name: "9. CIDR condition contains client IP",
			condition: &Condition{
				CIDRs: []string{"192.168.0.0/16", "10.0.0.0/8"},
			},
			ctx:  &authenticated,
			want: true,
		},
		{
			name: "10. state flags distinguish failed from successful authentication",
			condition: &Condition{
				MatchFailed: BoolPtr(true),
			},
			ctx: &FlowContext{
				OrgID:                "org-1",
				AuthenticationFailed: true,
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.condition.Matches(tt.ctx); got != tt.want {
				t.Fatalf("Matches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCondition_Validate(t *testing.T) {
	t.Parallel()

	t.Run("valid condition compiles regex and CIDR", func(t *testing.T) {
		t.Parallel()
		condition := &Condition{
			UserAgentRegexes: []string{`Mozilla.*`},
			CIDRs:            []string{"10.0.0.0/8", "192.168.1.1"},
		}
		if err := condition.Validate(); err != nil {
			t.Fatalf("Validate() unexpected error: %v", err)
		}
	})

	t.Run("invalid regex is reported", func(t *testing.T) {
		t.Parallel()
		condition := &Condition{UserAgentRegexes: []string{"["}}
		if err := condition.Validate(); err == nil {
			t.Fatal("Validate() expected error for invalid regex")
		}
	})

	t.Run("invalid CIDR is reported", func(t *testing.T) {
		t.Parallel()
		condition := &Condition{CIDRs: []string{"not-an-ip"}}
		if err := condition.Validate(); err == nil {
			t.Fatal("Validate() expected error for invalid CIDR")
		}
	})
}

func TestOrchestrator_SelectActions(t *testing.T) {
	t.Parallel()

	actions := []Action{
		testAction{id: "unconditional"},
		testAction{id: "org-scoped"},
		testAction{id: "other-org"},
	}

	registry := NewRegistry()
	if err := registry.SetCondition("org-1", "org-scoped", &Condition{Organizations: []string{"org-1"}}); err != nil {
		t.Fatal(err)
	}
	if err := registry.SetCondition("org-1", "other-org", &Condition{Organizations: []string{"org-other"}}); err != nil {
		t.Fatal(err)
	}

	flowCtx := &FlowContext{OrgID: "org-1"}
	selected, err := NewOrchestrator(registry).SelectActions(flowCtx, actions)
	if err != nil {
		t.Fatal(err)
	}

	gotIDs := make([]string, 0, len(selected))
	for _, action := range selected {
		gotIDs = append(gotIDs, action.GetID())
	}
	wantIDs := []string{"unconditional", "org-scoped"}
	if len(gotIDs) != len(wantIDs) {
		t.Fatalf("selected actions = %v, want %v", gotIDs, wantIDs)
	}
	for i, id := range wantIDs {
		if gotIDs[i] != id {
			t.Fatalf("selected actions = %v, want %v", gotIDs, wantIDs)
		}
	}
}

func TestOrchestrator_NilProviderKeepsLegacyFlow(t *testing.T) {
	t.Parallel()

	actions := []Action{testAction{id: "a"}, testAction{id: "b"}}
	selected, err := NewOrchestrator(nil).SelectActions(&FlowContext{OrgID: "org-1"}, actions)
	if err != nil {
		t.Fatal(err)
	}
	if len(selected) != len(actions) {
		t.Fatalf("selected %d actions, want all %d", len(selected), len(actions))
	}
}

func TestOrchestrator_ProviderError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("condition store unavailable")
	_, err := NewOrchestrator(failingProvider{err: wantErr}).
		SelectActions(&FlowContext{OrgID: "org-1"}, []Action{testAction{id: "a"}})
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}
}

func TestRegistry_SetCondition(t *testing.T) {
	t.Parallel()

	t.Run("empty condition removes stored condition", func(t *testing.T) {
		t.Parallel()
		registry := NewRegistry()
		if err := registry.SetCondition("org-1", "a", &Condition{AuthMethods: []string{"OTP"}}); err != nil {
			t.Fatal(err)
		}
		if err := registry.SetCondition("org-1", "a", nil); err != nil {
			t.Fatal(err)
		}
		condition, err := registry.ConditionForAction("org-1", "a")
		if err != nil {
			t.Fatal(err)
		}
		if condition != nil {
			t.Fatalf("condition = %v, want nil", condition)
		}
	})

	t.Run("organization and action IDs are required", func(t *testing.T) {
		t.Parallel()
		registry := NewRegistry()
		if err := registry.SetCondition("", "a", &Condition{}); err == nil {
			t.Fatal("expected error for empty organization ID")
		}
		if err := registry.SetCondition("org-1", "", &Condition{}); err == nil {
			t.Fatal("expected error for empty action ID")
		}
	})

	t.Run("remove organization clears all of its conditions", func(t *testing.T) {
		t.Parallel()
		registry := NewRegistry()
		if err := registry.SetCondition("org-1", "a", &Condition{Organizations: []string{"org-1"}}); err != nil {
			t.Fatal(err)
		}
		registry.RemoveOrganization("org-1")
		condition, err := registry.ConditionForAction("org-1", "a")
		if err != nil {
			t.Fatal(err)
		}
		if condition != nil {
			t.Fatalf("condition = %v, want nil", condition)
		}
	})
}

func TestNewFlowContext_FromAuthRequest(t *testing.T) {
	t.Parallel()

	authRequest := &domain.AuthRequest{
		RequestedOrgID:      "org-requested",
		UserOrgID:           "org-user",
		SelectedIDPConfigID: "idp-7",
		UserID:              "user-1",
		BrowserInfo: &domain.BrowserInfo{
			UserAgent: "Go-http-client",
			RemoteIP:  net.ParseIP("172.16.0.9"),
		},
	}
	httpRequest := httptest.NewRequest(http.MethodPost, "https://example.com/login", nil)

	flowCtx := NewFlowContext("org-requested", authRequest, httpRequest, "U2F", nil)

	if flowCtx.OrgID != "org-requested" || flowCtx.RequestedOrgID != "org-requested" || flowCtx.UserOrgID != "org-user" {
		t.Fatalf("unexpected organization fields: %+v", flowCtx)
	}
	if flowCtx.IDPID != "idp-7" || flowCtx.AuthMethod != "U2F" {
		t.Fatalf("unexpected IDP/auth method fields: %+v", flowCtx)
	}
	if !flowCtx.Authenticated || flowCtx.AuthenticationFailed {
		t.Fatalf("unexpected state flags: %+v", flowCtx)
	}
	if flowCtx.UserAgent != "Go-http-client" || !flowCtx.RemoteIP.Equal(net.ParseIP("172.16.0.9")) {
		t.Fatalf("unexpected browser fields: %+v", flowCtx)
	}
}

func TestNewFlowContext_FailedAuthentication(t *testing.T) {
	t.Parallel()

	flowCtx := NewFlowContext("org-1", &domain.AuthRequest{}, nil, "password", errors.New("invalid password"))
	if !flowCtx.AuthenticationFailed || flowCtx.Authenticated {
		t.Fatalf("unexpected state flags: %+v", flowCtx)
	}
}

func TestMatchesFlow_EndToEnd(t *testing.T) {
	t.Parallel()

	authRequest := &domain.AuthRequest{RequestedOrgID: "org-1"}
	httpRequest := httptest.NewRequest(http.MethodGet, "https://example.com/login", nil)
	httpRequest.RemoteAddr = "10.20.30.40:1234"

	condition := &Condition{
		Organizations: []string{"org-1"},
		AuthMethods:   []string{"password"},
		CIDRs:         []string{"10.20.30.0/24"},
	}
	if !MatchesFlow(condition, "org-1", authRequest, httpRequest, "password", nil) {
		t.Fatal("MatchesFlow() = false, want true")
	}

	if MatchesFlow(condition, "org-2", authRequest, httpRequest, "password", nil) {
		t.Fatal("MatchesFlow() = true for other organization, want false")
	}
}
