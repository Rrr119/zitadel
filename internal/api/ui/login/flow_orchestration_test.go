package login

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zitadel/zitadel/internal/domain"
	"github.com/zitadel/zitadel/internal/loginflow"
	"github.com/zitadel/zitadel/internal/query"
)

type staticConditionProvider struct {
	conditions map[string]*loginflow.Condition
}

func (p staticConditionProvider) ConditionForAction(_ string, actionID string) (*loginflow.Condition, error) {
	return p.conditions[actionID], nil
}

func TestLogin_filterFlowActions(t *testing.T) {
	t.Parallel()

	actions := []*query.Action{
		{ID: "unconditional", Name: "unconditional"},
		{ID: "org-scoped", Name: "org-scoped"},
		{ID: "other-org", Name: "other-org"},
	}

	provider := staticConditionProvider{
		conditions: map[string]*loginflow.Condition{
			"org-scoped": {Organizations: []string{"org-1"}},
			"other-org":  {Organizations: []string{"org-other"}},
		},
	}
	l := &Login{flowOrchestrator: loginflow.NewOrchestrator(provider)}

	authRequest := &domain.AuthRequest{RequestedOrgID: "org-1"}
	httpRequest := httptest.NewRequest(http.MethodPost, "https://example.com/login", nil)

	selected, err := l.filterFlowActions("org-1", authRequest, httpRequest, "password", nil, actions)
	if err != nil {
		t.Fatal(err)
	}

	gotIDs := make([]string, 0, len(selected))
	for _, action := range selected {
		gotIDs = append(gotIDs, action.ID)
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

func TestLogin_filterFlowActions_LegacyCompatibility(t *testing.T) {
	t.Parallel()

	actions := []*query.Action{
		{ID: "a"},
		{ID: "b"},
	}
	// The default registry carries no conditions, so every action is kept.
	l := &Login{flowOrchestrator: loginflow.NewOrchestrator(loginflow.NewRegistry())}

	selected, err := l.filterFlowActions(
		"org-1",
		&domain.AuthRequest{RequestedOrgID: "org-1"},
		httptest.NewRequest(http.MethodGet, "https://example.com/login", nil),
		"OTP",
		nil,
		actions,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(selected) != len(actions) {
		t.Fatalf("selected %d actions, want all %d (legacy flow must be unchanged)", len(selected), len(actions))
	}
}

func TestLogin_SetFlowConditionProvider(t *testing.T) {
	t.Parallel()

	l := &Login{}
	l.SetFlowConditionProvider(loginflow.NewRegistry())
	if l.flowOrchestrator == nil {
		t.Fatal("flow orchestrator must be initialized")
	}

	// Passing nil falls back to an empty registry instead of a nil provider.
	l.SetFlowConditionProvider(nil)
	if l.flowOrchestrator == nil {
		t.Fatal("flow orchestrator must not be nil after resetting the provider")
	}
}
