---
name: fixing-the-runtime
description: "Fix a bug or security issue in the engine-sdk 1.x runtime: the supervisor, the runtime commands, the host binary or the conformance suite, released as a patch. Use when a pull request touches command/, host/, supervise/, ctl/, requests/, responses/, dtos/, enums/, conformance/ or sdk.Version. Do not use for the image tools or workflows alone, and stop if the change needs a new command or field: the runtime is frozen."
---

# Fixing the runtime

`engine-sdk` 1.0 is a frozen runtime (`docs/adr/0009-engine-sdk-1-0-is-a-frozen-runtime.md`). Every Databasify engine image runs this binary, and the app sends the same runtime commands to images built on any 1.x release. Engine behaviour lives in the app's PHP drivers and Blade scripts, never here.

## What may change

- **Patches only:** bug and security fixes that keep every command, request field, reply field and error code as it is.
- **Not allowed:** a new command, a new or renamed field, a new error code, a changed meaning, or anything that names an engine. If a fix seems to need one, stop and open an issue for the maintainers.
- **Strict decoding stays.** An unknown request field is `invalid_request` (exit 64).
- **Host commands** (`host deploy`, `remove`, `tune`, `restore`) and forwarding stay compatible with every image built on `0.2.2` or any 1.x release.

## Where a fix goes

| Area | Files |
| --- | --- |
| A runtime command | its handler in `command/neutral.go` and its file in `command/` |
| The host binary | `host/` |
| The supervisor and `databasify ctl` | `supervise/`, `ctl/`, `units/` |
| A conformance check | `conformance/suite.go`, and the fake in `conformance/suite_test.go` |
| The version | `const Version` in `sdk.go`, written as plain `1.0.1`; only the git tag carries the `v` |

## Check it

```bash
go vet ./...
go test -race -count=1 ./...
```

- Add a test that fails without the fix.
- In the pull request, name the bug, the patch version it targets, and confirm that no command or field changed.
