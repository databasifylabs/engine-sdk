# 0002. Typed `databasify` commands are the only engine interface

- Status: Accepted
- Date: 2026-11-09

## Context

The app used to send Blade-rendered bash scripts over SSH and parse their text output. Engine logic was spread across the app, scripts were hard to test, and long operations held SSH connections open for minutes or hours.

## Decision

- The app runs `databasify <group> <verb>` on the server over SSH. It sends one JSON object on stdin and reads one JSON object on stdout.
- Exit `0` is a reply. Exit `1` is an error object `{"error":{"code","message","details"}}` with code `invalid_request`, `unsupported`, `not_found`, `already_exists`, `not_primary`, `busy`, `unavailable` or `internal`. Exit `64` means an invalid request and `70` an internal failure.
- Each server runs exactly one engine container. The app installs the official host binary (`cmd/databasify-host`) as `/usr/local/bin/databasify`. It forwards every well-formed `<group> <verb>` to that container and exits `69` when none is running.
- `host deploy`, `host remove`, `host tune` and `host restore` run on the host itself.
- Long work (`node ready`, `cluster enable`, `cluster rejoin`, `restore start`) returns an operation at once. The work runs in a transient unit. The app polls `operation status` and can `operation cancel`.
- There is no raw SQL or script endpoint: every feature is its own command.

## Consequences

- The app becomes a thin caller and never holds long SSH sessions.
- Engine behaviour is unit-tested in Go and live-tested in the image.
- Every new feature needs a command, a request type and a reply type.

## Alternatives considered

- Blade scripts over SSH, the previous approach: untyped, untested and duplicated per version line.
- `docker exec` from the app: ties the app to the container runtime, so the host binary forwards instead.
- A `sql run` escape hatch: rejected, because it bypasses validation.

## Evidence

- `sdk.go`
- `cmd/databasify-host/main.go`
- `command/command.go`
- `command/operation.go`
- `host/host.go`
