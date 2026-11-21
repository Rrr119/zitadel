package loginflow

import (
	"net"
	"net/http"

	"github.com/zitadel/zitadel/internal/domain"
)

// FlowContext contains the runtime information of a login flow against which
// conditions are evaluated. It is intentionally transport independent so that
// the same orchestration is used by the internal and external authentication,
// pre creation and post creation triggers.
type FlowContext struct {
	// OrgID is the resource owner for which the flow is executed. It is the
	// requested org, the user org or the instance default org, in that order.
	OrgID string

	// RequestedOrgID is the organization explicitly requested by the client
	// (scope or domain discovery), if any.
	RequestedOrgID string

	// UserOrgID is the organization the authenticating user belongs to, if any.
	UserOrgID string

	// AuthMethod is the method used for the current authentication, e.g.
	// "password", "OTP", "U2F". It is empty for creation flows.
	AuthMethod string

	// IDPID is the identity provider configuration ID selected for an external
	// authentication, if any.
	IDPID string

	// UserAgent is the User-Agent header of the current HTTP request.
	UserAgent string

	// RemoteIP is the client IP of the current HTTP request.
	RemoteIP net.IP

	// Authenticated reports whether the user is already authenticated when the
	// action is evaluated.
	Authenticated bool

	// AuthenticationFailed reports whether the current authentication attempt
	// failed before the action is evaluated.
	AuthenticationFailed bool
}

// NewFlowContext builds a flow context for the given organization, auth
// request and HTTP request. authMethod may be empty on creation flows and
// authenticationError may be nil on a successful authentication.
func NewFlowContext(
	orgID string,
	authRequest *domain.AuthRequest,
	httpRequest *http.Request,
	authMethod string,
	authenticationError error,
) *FlowContext {
	flowCtx := &FlowContext{
		OrgID:                orgID,
		AuthMethod:           authMethod,
		AuthenticationFailed: authenticationError != nil,
	}
	if authRequest != nil {
		flowCtx.RequestedOrgID = authRequest.RequestedOrgID
		flowCtx.UserOrgID = authRequest.UserOrgID
		flowCtx.IDPID = authRequest.SelectedIDPConfigID
		if authRequest.UserID != "" {
			flowCtx.Authenticated = true
		}
		if authRequest.BrowserInfo != nil {
			flowCtx.UserAgent = authRequest.BrowserInfo.UserAgent
			flowCtx.RemoteIP = authRequest.BrowserInfo.RemoteIP
		}
	}
	if httpRequest != nil {
		if flowCtx.UserAgent == "" {
			flowCtx.UserAgent = httpRequest.UserAgent()
		}
		if flowCtx.RemoteIP == nil {
			flowCtx.RemoteIP = remoteIPFromRequest(httpRequest)
		}
	}
	return flowCtx
}
