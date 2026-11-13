# 0009. engine-sdk 1.0 is a frozen runtime

- Status: Accepted
- Date: 2026-11-13

## Context

Engine behaviour moved out of Go into the Databasify app: each engine is a PHP driver that runs hardened Blade scripts in the container through `runtime run`, `task start` and `file put`. The app declares each engine's facts (ports, reserved names, password encryptions, host profile) itself and sends the host profile with `host deploy`. Backups, log shipping, migration copies and cluster watchers are Bash jobs the app writes to the node.

That left the Go engine contract, the typed engine commands and the engine helper packages with no caller. Every new engine or feature would still have meant an SDK release and an image rebuild. The owner wants the SDK final, so that it changes only for bug fixes.

## Decision

- **One generic binary.** `cmd/databasify` is the supervisor, the runtime and the host binary in one. `sdk.Run()` takes no engine. The engine is `DATABASIFY_ENGINE`, set from the image's `io.databasify.engine` label.
- **No engine contract.** `contract/`, the command router and every engine command are gone, with the engine helpers only engines used (`settings`, `rclone`, `testtls`, `version` and the TLS writer). A well-formed command outside the runtime answers `unsupported`.
- **The runtime is the whole interface:**
  - `engine info` and `runtime info`;
  - `runtime run`;
  - `task start`, `status`, `log` and `stop`;
  - `file get` and `put`;
  - `service start`, `stop`, `restart`, `status` and `reload`;
  - `logs read` and `forward`;
  - on the host, `host deploy`, `remove`, `tune` and `restore`.
- **The host profile comes in the request.** `host deploy` and `host tune` take `hostProfile` and never ask the image for one.
- **An engine image** is the official database image, its tools, and the binary copied from `ghcr.io/databasifylabs/engine-sdk` by digest. It has no Go code of its own. Its `versions.json` pins `sdk`, and `tools/images.sh` reads the pin from there.
- **Versioning.** `1.0.0` is the last feature release. Later releases are patches: bug and security fixes, with no new commands or fields. A new engine or engine feature ships as a Databasify driver and Blade scripts.
- **Conformance** checks the runtime only.

## Consequences

- An engine image changes only when its database or a tool moves. Engine fixes ship with the app.
- Outside authors write a PHP driver and Blade scripts against a runtime that will not move under them.
- The app must keep every engine's behaviour in its own scripts, with no engine command to fall back on.
- Images built on `0.2.2` keep working, since they answer the same runtime commands; their engine commands go unused.

## Alternatives considered

- **Keep the contract for outside Go engines:** two ways to write an engine, and every feature would still need an SDK release.
- **Keep `0.x` and break freely:** outside authors could never rely on the runtime.

## Evidence

- `sdk.go`
- `cmd/databasify/main.go`
- `command/command.go`
- `command/neutral.go`
- `host/commands.go`
- `host/engine.go`
- `requests/host_deploy.go`
- `requests/host_tune.go`
- `conformance/suite.go`
- `tools/images.sh`
