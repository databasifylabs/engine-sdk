# 0006. Engines declare their host profile

- Status: Superseded by 0009
- Date: 2026-11-11

## Context

`host tune` sets kernel settings on the server for the database it runs, such as swappiness, `somaxconn` and transparent huge pages. The host binary used to hold a table of these settings per engine name. The official host binary runs every engine, including engines written outside the team, so it cannot know each engine, and the SDK names no engine ([0001](0001-the-public-engine-sdk.md)).

## Decision

- `contract.Engine` has `HostProfile() HostProfile`: the sysctl `Settings` to set, the `Floors` (a minimum: the host keeps the larger of the current value and the floor) and whether to disable transparent huge pages. The zero value tunes nothing.
- The quick command `engine host-profile` (no request fields) replies with `responses.EngineHostProfile` (`settings`, `floors`, `disableTransparentHugepages`). Every engine answers it; it is never `unsupported`.
- `host tune`, and the tune at the end of `host deploy`, read the engine name from `engine info` and the profile from `engine host-profile` in the running engine container. They write `/etc/sysctl.d/99-<engine>.conf`, keep the original values in `/etc/databasify/sysctl-originals/<engine>`, and apply them with `sysctl --system`. With transparent huge pages disabled, they install `disable-thp.service` and keep the original modes in `/etc/databasify/thp-originals`. A zero profile changes nothing on the host. A container that does not answer `engine host-profile` fails `host tune` with an error naming its container and image; it is never read as "no tuning".
- `host restore` puts the originals back.
- The app never calls `engine host-profile`; only the host binary does.

## Consequences

- Each engine keeps its kernel tuning next to its own code, and an outside engine can tune the host without a host binary release.
- An engine that returns the zero profile runs on an untuned host.

## Alternatives considered

- **A per-engine table in the host binary:** the SDK would name engines, and outside engines would get no tuning.
- **An image label with the profile:** `host tune` and `host restore` run without an image in hand, and the floors would need the same rules encoded in a label.

## Evidence

- `contract/engine.go`
- `command/neutral.go`
- `responses/engine_host_profile.go`
- `host/tuning.go`
- `host/deploy.go`
