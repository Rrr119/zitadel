// Package orgpolicy provides organization scoped UI branding and login
// policy configuration with instance default inheritance.
//
// ZITADEL's established IAM model defines label policies (UI branding) and
// login policies at two ownership levels:
//
//   - the instance owns the default policy, which applies to every
//     organization that has not customized the setting;
//   - an organization may own a custom policy that overrides the instance
//     default. When no organization policy exists, resolution returns the
//     instance default and marks the result as inherited (the equivalent of
//     Default == true on domain.LabelPolicy and domain.LoginPolicy).
//
// This package captures that ownership and inheritance contract in a small,
// self contained and concurrency safe configuration service. Every read and
// write is keyed by the tuple (instanceID, organizationID), which makes
// policies of different organizations — and instances — strictly isolated.
//
// The iamcompat subpackage maps these types to the canonical
// internal/domain.LabelPolicy and internal/domain.LoginPolicy types so the
// package can coexist with the existing event sourced IAM stack.
package orgpolicy
