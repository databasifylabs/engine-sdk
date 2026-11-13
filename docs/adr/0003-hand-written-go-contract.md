# 0003. The contract is hand-written Go in Laravel-style packages

- Status: Superseded by 0009
- Date: 2026-11-09

## Context

The first contract was JSON Schemas with a code generator and a breaking-change checker in a separate repository. It was more machinery than the project needed, and its generated types were hard to read.

## Decision

- **`contract`** holds the `Engine` interface and one interface per group: `Nodes`, `Accounts`, `Cluster`, `Archive`, `Migration`, `Pools` and `Config`. It also holds `Capabilities`, `HostProfile`, the `Unsupported*` stand-ins and the command catalog.
- **Data types:**
  - `requests` and `responses` hold one hand-written struct per command, named without a suffix (`requests.UserCreate`).
  - `dtos` holds shared data and `enums` holds enums with `Valid()`.
  - `support` holds strict JSON: exact keys, required fields, `pattern`, `min` and `max` tags. Empty lists encode as `[]` and empty maps as `{}`.
- **Strict types:** no `any` outside generic constraints and driver boundaries.
- **Versioning:** there is one version, `sdk.Version` ([0001](0001-the-public-engine-sdk.md)).
- **Unsupported features:** a group an engine doesn't offer returns its `Unsupported*` stand-in, and `engine capabilities` lists only what is implemented.

## Consequences

- Adding an engine means implementing the interfaces; the compiler finds every missing method.
- The app maps replies into its own DTOs; there are no generated PHP classes.

## Alternatives considered

- JSON Schema with generated Go and PHP: removed as unneeded machinery.
- A separate `engine-api` repository: deleted.
- Laravel Zero for the wrapper: rejected in favour of keeping it all in Go.

## Evidence

- `contract/engine.go`
- `support/json.go`
- `command/routes.go`
