package login

import (
	"net/http"

	"github.com/zitadel/zitadel/internal/domain"
	"github.com/zitadel/zitadel/internal/loginflow"
	"github.com/zitadel/zitadel/internal/query"
)

// SetFlowConditionProvider replaces the condition provider used to orchestrate
// the organization scoped login flows. Passing nil restores the default empty
// registry, where every active action of a flow trigger is executed.
func (l *Login) SetFlowConditionProvider(provider loginflow.ConditionProvider) {
	if provider == nil {
		provider = loginflow.NewRegistry()
	}
	l.flowOrchestrator = loginflow.NewOrchestrator(provider)
}

// selectFlowActions resolves the active actions of a flow trigger and keeps
// only the ones whose organization scoped condition matches the current flow.
// Actions without a condition are always returned, preserving the behavior of
// existing flow configurations.
func (l *Login) selectFlowActions(
	resourceOwner string,
	flowType domain.FlowType,
	triggerType domain.TriggerType,
	authRequest *domain.AuthRequest,
	httpRequest *http.Request,
	authMethod string,
	authenticationError error,
) ([]*query.Action, error) {
	ctx := httpRequest.Context()
	triggerActions, err := l.query.GetActiveActionsByFlowAndTriggerType(ctx, flowType, triggerType, resourceOwner)
	if err != nil {
		return nil, err
	}
	return l.filterFlowActions(resourceOwner, authRequest, httpRequest, authMethod, authenticationError, triggerActions)
}

// filterFlowActions keeps the actions whose organization scoped condition
// matches the current flow. It is split from the database lookup so that the
// orchestration can be tested without a running query layer. Actions without a
// condition are always kept, preserving the legacy flow behavior.
func (l *Login) filterFlowActions(
	resourceOwner string,
	authRequest *domain.AuthRequest,
	httpRequest *http.Request,
	authMethod string,
	authenticationError error,
	triggerActions []*query.Action,
) ([]*query.Action, error) {
	if l.flowOrchestrator == nil {
		// Defensive default for Login values created without CreateLogin:
		// without a provider every action is selected, matching the legacy flow.
		l.flowOrchestrator = loginflow.NewOrchestrator(nil)
	}
	flowCtx := loginflow.NewFlowContext(resourceOwner, authRequest, httpRequest, authMethod, authenticationError)
	actions := make([]loginflow.Action, len(triggerActions))
	for i, action := range triggerActions {
		actions[i] = queryAction{action}
	}
	selected, err := l.flowOrchestrator.SelectActions(flowCtx, actions)
	if err != nil {
		return nil, err
	}

	selectedActions := make([]*query.Action, len(selected))
	for i, action := range selected {
		selectedActions[i] = action.(queryAction).Action
	}
	return selectedActions, nil
}

// queryAction adapts *query.Action to the minimal loginflow.Action interface.
type queryAction struct {
	*query.Action
}

func (a queryAction) GetID() string {
	return a.Action.ID
}
