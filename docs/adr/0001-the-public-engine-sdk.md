# 0001. The public engine SDK

- Status: Accepted
- Date: 2026-11-11

## Context

Every engine lived in one `databasifylabs/engines` module with the framework under `internal/`, so nothing outside that repository could build an engine. Engines are meant to be written by outside authors too, each in their own repository, and the team's six engines are moving into one repository each.

## Decision

- `github.com/databasifylabs/engine-sdk` holds everything no single engine owns: the contract (`contract`), the request, reply, shared data and enum types (`requests`, `responses`, `dtos`, `enums`), strict JSON (`support`), the command router and operations (`command`), the supervisor (`supervise`) and `ctl`, the settings and process manifest builders (`settings`, `manifest`), the host commands and forwarding (`host`), the node filesystem helpers (`nodefs`), log shipping (`logship`), the S3 copy helper (`rclone`), `step`, `layout`, `units`, `version`, `testtls` and the conformance suite (`conformance`).
- Packages live at the module root so any module can import them. Nothing in the SDK names a specific engine.
- An engine's `main` is one call: `os.Exit(sdk.Run(engine))`. `sdk.Run` does what the shared binary did, with the engine passed in.
- `cmd/databasify-host` is the official host binary: the `host` commands, `version`, and forwarding of every well-formed `<group> <verb>` (two lowercase words, such as `user set-password`) into the engine container, known to its own catalog or not. Anything else is a usage error.
- Inside the container, `sdk.Run` answers a well-formed `<group> <verb>` that is not in its catalog with the `unsupported` error object and exit `1`, so an engine on an older SDK answers a newer command with `unsupported`.
- The SDK follows semantic versioning. Its first release is `0.2.0` (`sdk.Version`) and stays at `0.x`, where any minor may break, until the team's engines have proven the contract; then `1.0`. Within a major, releases only add commands, optional fields and reply fields.
- `engine info` reports `sdk.Version` in its `sdk` field, and every image carries it in the `io.databasify.sdk` label.
- `conformance.Run(t, runner)` runs the suite against a running engine, so an author runs it on their own image.

## Consequences

- Outside authors build an engine against a versioned, importable framework, and an SDK fix reaches an engine when its author rebuilds it.
- A change to the contract is a release of this module and a rebuild of every engine, instead of one commit in one repository.
- The host binary and the engines are released separately; a pinned host binary must deploy every image in the app's SDK range.

## Alternatives considered

- **Keep one repository with every engine:** outside authors could not build on it, and every engine change rebuilt every image.
- **Keep the packages under `internal/` and vendor them:** a copy per engine drifts and has no version to check.

## Evidence

- `sdk.go`
- `cmd/databasify-host/main.go`
- `contract/engine.go`
- `conformance/suite.go`
