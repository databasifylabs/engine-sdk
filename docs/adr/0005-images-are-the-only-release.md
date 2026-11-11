# 0005. Images are the only release

- Status: Accepted
- Date: 2026-11-11

## Context

The engines repository once published the binary alone as `databasify-wrapper` and installed it over every engine with `host wrapper-update`. A tag that moves means a node can run a binary no image ever shipped, so what runs is no longer what was built, scanned and smoke-tested. `host deploy` also overwrote the host binary with the one inside the image.

## Decision

- An engine image is the only thing an engine publishes. There is no `databasify-wrapper` image and no `host wrapper-update`. A fix reaches nodes through a new image and `host deploy`.
- `host deploy` does not touch the host binary. The app installs the official host binary on its own.
- Every image carries the SDK version it was built on in the `io.databasify.sdk` label. The host binary finds the engine container by `io.databasify.engine`, which every image carries.

## Consequences

- What runs on a node is exactly what an image build scanned and smoke-tested.
- A change to the SDK reaches nodes only through an engine rebuild and a rolling `host deploy`, which restarts each node.

## Alternatives considered

- **Keep the wrapper image for hotfixes:** a second release channel that lets a node run unbuilt code, which is what this removes.

## Evidence

- `host/commands.go`
- `host/host.go`
