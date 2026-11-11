---
name: changing-the-contract
description: "Change the engine command contract in engine-sdk: add a command, a request or reply field, a shared type, a host command, or change the conformance suite, following the SDK's versioning rules. Use when a pull request touches contract/, requests/, responses/, dtos/, enums/, command/, host/, conformance/ or sdk.Version. Do not use for the image tools or workflows alone."
---

# Changing the engine contract

Every Databasify engine, the team's and every community engine, builds on this module and answers the commands in `contract/commands.go`. The Databasify app sends those commands to engines built on any SDK version inside its supported range. So a change here must keep working against engines built on older releases of the same major.

## The versioning rules

- The SDK follows semantic versioning. Before `1.0`, any minor release may break; from `1.0` on, the rules below apply within a major.
- **Within a major, only add:**
  - a new command;
  - a new optional request field;
  - a new reply field;
  - a new value a reply may carry, when the app already treats unknown values safely.
- **An engine that lacks a newer command answers `unsupported`** (exit 1). `command.dispatch` does this for any well-formed `<group> <verb>` outside the catalog, so add the command to `contract.Commands` and route it in `command/routes.go`. Never make an older engine fail differently.
- **Request decoding is strict.** An unknown field is `invalid_request` (exit 64), so an engine on an older release rejects a field it doesn't know. A new request field must be optional, its zero value must mean the old behaviour, and the change must say that callers leave it out for engines that predate it.
- **Breaking is a new major.** Removing or renaming a command or field, changing a type or meaning, or making an optional field required all count as breaking. A new major ships with an upgrade guide in `docs/`.
- **Host commands follow the same rules.** The app pins one host binary (`cmd/databasify-host`) that must deploy every image in its SDK range. `host deploy`, `host remove`, `host tune` and `host restore`, and the forwarding into the container, stay compatible with every engine of the major. Forwarding passes any well-formed `<group> <verb>` through, so it never needs a new entry for a new command.

## Where a change goes

| Change | Files |
| --- | --- |
| A command | `contract/commands.go` (name, group, verb, quick or long), the group interface in `contract/engine.go`, the `Unsupported*` stand-in, the route in `command/routes.go`, request and reply types in `requests/` and `responses/` |
| A field | the type in `requests/`, `responses/` or `dtos/`, with its JSON name in camelCase |
| A conformance check | `conformance/suite.go`. It tests only what `engine capabilities` lists, plus `engine capabilities`, `engine host-profile`, `engine info` and `node status` |
| The version | `const Version` in `sdk.go`. It is written as plain `0.2.0`; only the git tag carries the `v` |

Add an ADR in `docs/adr/` for a lasting decision, and update `README.md` when a rule or a public name changes.

## Check it

```bash
go vet ./...
go test -race -count=1 ./...
```

- Add a test for the new behaviour, and one showing that an engine without it still answers `unsupported` or ignores the absent field.
- The conformance suite's own tests (`conformance/suite_test.go`) run against a fake engine. Extend the fake when you add a check.
- In the pull request, say whether the change is additive or breaking, which SDK version it targets, and what the Databasify app must do to use it. Open an issue first for a new command.
