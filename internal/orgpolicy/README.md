# Organization Branding & Policy Inheritance

Organization scoped UI branding (ZITADEL *label policy*) and authentication
policy (*login policy*) with instance default inheritance and strict
multi organization isolation.

## How it fits the existing IAM model

ZITADEL already models these policies at two ownership levels
(`internal/domain/policy_label.go`, `internal/domain/policy_login.go`,
resolved by `internal/query/label_policy.go` and
`internal/query/login_policy.go`):

- the **instance** owns a default policy;
- an **organization** may own a custom policy, otherwise the instance default
  is returned with `Default == true`.

This package reproduces that contract as a self contained domain service and
adds explicit inheritance provenance:

| Concept            | This package                         | Existing ZITADEL equivalent        |
| ------------------ | ------------------------------------ | ---------------------------------- |
| UI branding        | `Branding`                           | `domain.LabelPolicy`               |
| Login policy       | `LoginPolicy`                        | `domain.LoginPolicy`               |
| Inherited marker   | `Resolved[T].Inherited` / `.Source`  | `LabelPolicy.Default` / `LoginPolicy.Default` |
| Instance default   | `Service.GetDefault*` / `UpdateDefault*` | `Queries.DefaultLoginPolicy`, admin defaults |
| Org get/update/reset | `Service.Get*` / `Update*` / `Reset*` | Management `Get/Update/ResetLoginPolicy`, `...LabelPolicy` |
| Ownership key      | `(instanceID, orgID)`                | `instance_id` + `resource_owner`   |

The `iamcompat` subpackage provides lossless bidirectional mappers to the
canonical `domain.LabelPolicy` and `domain.LoginPolicy` types. Mapped
inherited policies carry `Default: true` and the correct
`ObjectRoot.InstanceID` / `ResourceOwner`, so the model coexists with the
event sourced command/query stack and existing APIs.

## Inheritance rules

- An organization without an override resolves the **instance default** and is
  marked `Source = instance_default`, `Inherited = true`.
- `PatchBranding` / `PatchLoginPolicy` merge optional fields over the
  currently effective policy, so unspecified fields keep their inherited
  values; the merged result becomes the organization override.
- `ResetBranding` / `ResetLoginPolicy` delete the organization override so the
  organization inherits the instance default again.
- Colors use the same `#rgb` / `#rrggbb` validation as the core label policy.

## Isolation

Every repository operation is keyed by `instanceID` and `orgID`. Policies and
mutations of one organization (or instance) can never be read or modified
through another. `Store` is safe for concurrent use; a `Repository`
interface allows a relational/event backed adapter to replace the in memory
reference implementation.

## Three organization acceptance scenario

`TestThreeOrganizations_*` builds one instance with exactly three
organizations:

- `org-acme`: fully custom brand and strict login policy (MFA enforced,
  registration disabled).
- `org-beta`: a single brand color, theme mode and the registration flag
  overridden; every other field is asserted to equal the instance default.
- `org-gamma`: no override; the whole resolved configuration is asserted to
  equal the instance default and be marked inherited.

The isolation test then mutates Acme and Beta and proves Gamma is untouched,
Beta only shows its own values, Acme stores exactly its own changes, and
resetting Acme does not affect the others.

## Run the tests

```bash
bash internal/orgpolicy/run_tests.sh
# or directly with a Go 1.26+ toolchain:
go test ./internal/orgpolicy/... -count=1 -v
```

The core package depends only on the Go standard library; the compatibility
tests additionally import `internal/domain`.
