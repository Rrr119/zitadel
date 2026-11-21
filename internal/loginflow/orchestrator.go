package loginflow

import (
	"net"
	"net/http"

	"github.com/zitadel/zitadel/internal/domain"
)

// Action is the subset of a configured action the orchestrator needs. The
// interface is satisfied by *query.Action and keeps the package free of
// database dependencies.
type Action interface {
	// GetID returns the identifier of the action.
	GetID() string
}

// ConditionProvider resolves the optional condition of an action for an
// organization. Implementations are expected to return a nil condition without
// an error when the action has no condition configured, which preserves the
// legacy behavior of running every active action of a flow trigger.
type ConditionProvider interface {
	// ConditionForAction returns the condition configured for the action in
	// the given organization, or nil when no condition exists.
	ConditionForAction(orgID, actionID string) (*Condition, error)
}

// Orchestrator selects the actions of an organization scoped flow that have to
// run for the current flow context. It is safe for concurrent use as long as
// the underlying ConditionProvider is.
type Orchestrator struct {
	provider ConditionProvider
}

// NewOrchestrator creates an orchestrator. A nil provider means that no
// action carries a condition, so every action is selected and existing flows
// behave exactly as before.
func NewOrchestrator(provider ConditionProvider) *Orchestrator {
	return &Orchestrator{provider: provider}
}

// SelectActions keeps the order of the given actions and returns the ones
// whose condition matches the flow context. Actions without a condition are
// always selected. The first resolution or validation error is returned.
func (o *Orchestrator) SelectActions(ctx *FlowContext, actions []Action) ([]Action, error) {
	selected := make([]Action, 0, len(actions))
	for _, action := range actions {
		if o == nil || o.provider == nil {
			selected = append(selected, action)
			continue
		}
		condition, err := o.provider.ConditionForAction(ctx.OrgID, action.GetID())
		if err != nil {
			return nil, err
		}
		if condition == nil {
			selected = append(selected, action)
			continue
		}
		if err := condition.Validate(); err != nil {
			return nil, err
		}
		if condition.Matches(ctx) {
			selected = append(selected, action)
		}
	}
	return selected, nil
}

// Select is a package level convenience around NewOrchestrator. A nil
// provider returns the actions unchanged.
func Select(provider ConditionProvider, ctx *FlowContext, actions []Action) ([]Action, error) {
	return NewOrchestrator(provider).SelectActions(ctx, actions)
}

// MatchesFlow evaluates a single condition against the flow described by the
// given organization, auth request and HTTP request. It is the entry point for
// callers that resolve conditions themselves instead of using a provider.
func MatchesFlow(condition *Condition, orgID string, authRequest *domain.AuthRequest, httpRequest *http.Request, authMethod string, authenticationError error) bool {
	flowCtx := NewFlowContext(orgID, authRequest, httpRequest, authMethod, authenticationError)
	return condition.Matches(flowCtx)
}

func remoteIPFromRequest(r *http.Request) net.IP {
	if r == nil {
		return nil
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	return net.ParseIP(host)
}
