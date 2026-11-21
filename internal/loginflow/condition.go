// Package loginflow provides organization scoped, condition based action
// orchestration for the login flows. It complements the existing flow
// triggers (see internal/domain/flow.go): actions are still resolved through
// the regular flow/trigger configuration and may additionally declare a
// condition that is evaluated against the current login flow context.
//
// An action without a condition is always selected, so existing flow
// configurations keep working without any change.
package loginflow

import (
	"net"
	"regexp"
	"slices"
)

// Condition describes when an action of an organization scoped login flow is
// triggered. Every populated field has to match (AND). The entries of slice
// fields are alternatives (OR), e.g. an action is allowed for multiple
// organizations or authentication methods.
type Condition struct {
	// Organizations restricts the action to the given organization IDs
	// (the resolved resource owner of the flow, see FlowContext.OrgID).
	Organizations []string

	// AuthMethods restricts the action to the given authentication methods,
	// e.g. "password", "OTP" or "U2F". Creation flows use an empty method, so
	// include an empty string to target them.
	AuthMethods []string

	// IDPIDs restricts the action to external authentications with the given
	// identity provider configuration IDs.
	IDPIDs []string

	// UserAgentRegexes are matched against the request user agent. The action
	// is triggered when one of the expressions matches.
	UserAgentRegexes []string

	// CIDRs restricts the action to client remote addresses contained in one
	// of the given CIDR notations. Single IP addresses are allowed as well.
	CIDRs []string

	// MatchAuthenticated controls whether the action runs for authenticated
	// users. nil does not restrict on the authentication state.
	MatchAuthenticated *bool

	// MatchFailed controls whether the action runs after a failed
	// authentication. nil does not restrict on the failure state.
	MatchFailed *bool
}

// IsEmpty reports whether the condition does not restrict the action at all.
// Empty conditions always match and represent the legacy flow behavior.
func (c *Condition) IsEmpty() bool {
	if c == nil {
		return true
	}
	return len(c.Organizations) == 0 &&
		len(c.AuthMethods) == 0 &&
		len(c.IDPIDs) == 0 &&
		len(c.UserAgentRegexes) == 0 &&
		len(c.CIDRs) == 0 &&
		c.MatchAuthenticated == nil &&
		c.MatchFailed == nil
}

// Validate compiles the regular expressions and CIDR notations of the
// condition so that misconfiguration is reported when the condition is
// registered instead of during a login.
func (c *Condition) Validate() error {
	if c == nil {
		return nil
	}
	for _, expr := range c.UserAgentRegexes {
		if _, err := regexp.Compile(expr); err != nil {
			return err
		}
	}
	for _, cidr := range c.CIDRs {
		if _, _, err := net.ParseCIDR(cidr); err != nil {
			ip := net.ParseIP(cidr)
			if ip == nil {
				return err
			}
		}
	}
	return nil
}

// Matches evaluates the condition against the flow context. A nil or empty
// condition matches every context, which keeps unconfigured flows compatible.
func (c *Condition) Matches(ctx *FlowContext) bool {
	if c.IsEmpty() {
		return true
	}
	if ctx == nil {
		return false
	}
	if len(c.Organizations) > 0 && !matchOrg(c.Organizations, ctx) {
		return false
	}
	if len(c.AuthMethods) > 0 && !slices.Contains(c.AuthMethods, ctx.AuthMethod) {
		return false
	}
	if len(c.IDPIDs) > 0 && !slices.Contains(c.IDPIDs, ctx.IDPID) {
		return false
	}
	if len(c.UserAgentRegexes) > 0 && !matchUserAgent(c.UserAgentRegexes, ctx.UserAgent) {
		return false
	}
	if len(c.CIDRs) > 0 && !matchCIDR(c.CIDRs, ctx.RemoteIP) {
		return false
	}
	if c.MatchAuthenticated != nil && *c.MatchAuthenticated != ctx.Authenticated {
		return false
	}
	if c.MatchFailed != nil && *c.MatchFailed != ctx.AuthenticationFailed {
		return false
	}
	return true
}

func matchOrg(orgs []string, ctx *FlowContext) bool {
	return slices.Contains(orgs, ctx.OrgID)
}

func matchUserAgent(exprs []string, userAgent string) bool {
	if userAgent == "" {
		return false
	}
	for _, expr := range exprs {
		matched, err := regexp.MatchString(expr, userAgent)
		if err == nil && matched {
			return true
		}
	}
	return false
}

func matchCIDR(cidrs []string, remoteIP net.IP) bool {
	if remoteIP == nil {
		return false
	}
	for _, entry := range cidrs {
		if _, network, err := net.ParseCIDR(entry); err == nil {
			if network.Contains(remoteIP) {
				return true
			}
			continue
		}
		if ip := net.ParseIP(entry); ip != nil && ip.Equal(remoteIP) {
			return true
		}
	}
	return false
}

// BoolPtr returns a pointer to the given bool. It is a small helper for
// building conditions in tests and registrations.
func BoolPtr(value bool) *bool {
	return &value
}
