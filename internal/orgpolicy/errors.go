package orgpolicy

import (
	"errors"
	"fmt"
	"strings"
)

// Sentinel errors used by the service and store. Callers can use errors.Is
// to detect them.
var (
	// ErrInvalidInstanceID is returned when an empty instance id is used.
	ErrInvalidInstanceID = errors.New("orgpolicy: instance id must not be empty")
	// ErrOrganizationNotFound is returned when an operation targets an
	// organization that has not been registered on the instance.
	ErrOrganizationNotFound = errors.New("orgpolicy: organization not found")
	// ErrOrganizationExists is returned when registering a duplicate
	// organization id on the same instance.
	ErrOrganizationExists = errors.New("orgpolicy: organization already exists")
	// ErrInvalidPolicy is returned when a policy fails validation.
	ErrInvalidPolicy = errors.New("orgpolicy: invalid policy")
)

// FieldError describes a single invalid field of a policy.
type FieldError struct {
	// Field is the dotted policy field path, for example "light.primaryColor".
	Field string
	// Message is a human readable explanation.
	Message string
}

func (e FieldError) String() string {
	return e.Field + ": " + e.Message
}

// ValidationError aggregates every invalid field of a policy so callers can
// fix all problems in one pass. It wraps ErrInvalidPolicy.
type ValidationError struct {
	Issues []FieldError
}

func (e *ValidationError) Error() string {
	parts := make([]string, 0, len(e.Issues))
	for _, issue := range e.Issues {
		parts = append(parts, issue.String())
	}
	return fmt.Sprintf("%s: %s", ErrInvalidPolicy, strings.Join(parts, "; "))
}

// Unwrap allows errors.Is(err, ErrInvalidPolicy).
func (e *ValidationError) Unwrap() error {
	return ErrInvalidPolicy
}

func validationError(issues []FieldError) error {
	if len(issues) == 0 {
		return nil
	}
	return &ValidationError{Issues: issues}
}
