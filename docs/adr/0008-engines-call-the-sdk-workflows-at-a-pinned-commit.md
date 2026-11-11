# 0008. Engines call the SDK workflows at a pinned commit

- Status: Accepted
- Date: 2026-11-11

## Context

[0007](0007-engines-publish-through-the-sdk-workflows.md) had engine repositories call the reusable workflows at the SDK tag their `go.mod` requires. On 2026-11-11 image builds on GitHub's runners stopped resolving `archive.ubuntu.com` inside BuildKit, and the fix to the release logic could only reach the engines through a new SDK tag. The SDK is stable and is not tagged for workflow-only changes, because every tag also publishes a host binary.

Every engine, the team's and a plugin author's, lives in its own repository and publishes images the same way: check the pins, hash what an image is built from, build and smoke-test both platforms, scan, publish, and refresh the pins weekly. Copying that CI into each repository lets the copies drift, and an author would maintain release logic that is not theirs.

## Decision

- The SDK ships `engine-images.yml` and `engine-refresh.yml` as reusable workflows with the inputs `engine` and `versions-file`. An engine repository calls them at a full 40-character SDK commit SHA (`engine-images.yml@<sha>`), never a tag or a branch; it holds `images/<engine>/` and a `versions.json` with `registry`, `sdk`, `go` and its own lines.
- `tools/images.sh` runs from the engine repository root, reads that repository's `versions.json` and `images/`, and takes its helper scripts from the SDK checkout the workflow makes at the `go.mod` version.
- An image's content hash covers the engine's Go sources, `go.mod` and `go.sum` (so the SDK version), its image files and its pins. Every `replace` to a local path (the SDK, or a sibling engine such as `engine-mysql` for MariaDB) must be written `../<dir>`; its Go sources join the hash and are staged into the build context `modules`, which every build passes (empty without replacements) and every engine Dockerfile copies to `/src/`. A missing replacement fails the hash. Dropping the replacements once the modules are tagged needs no Dockerfile change.
- The image builder (BuildKit, `docker-container` driver) resolves names through `1.1.1.1` and `8.8.8.8`, set in a `buildkitd` config, not through the runner's resolver.
- `engine-images.yml` runs the engine's `go vet` and `go test -race` before it builds, so an engine repository needs no Go workflow of its own.
- `check` refuses a `versions.json` `sdk` that differs from the version `go.mod` requires.
- `<release>`, `<line>` and `latest` move only on the caller's default branch; every other branch publishes `<release>-<sha7>` only.

## Consequences

- A fix to the release logic reaches an engine when its two caller workflows move to the commit carrying it; no SDK tag and no host binary release is needed.
- The workflow commit and the SDK version in `go.mod` can differ; the tools are still checked out at the `go.mod` version.
- Engine repositories keep two short caller workflows; the build and publish logic has one home.
- The caller's images workflow must be named `images.yml`: the refresh dispatches it on the pull request branch.
- Only tagged SDK versions can be checked out for the tools; a pseudo-version in `go.mod` does not resolve.

## Alternatives considered

- **A template that copies the workflows into each engine:** the copies drift and every fix is a pull request per engine.
- **Composite actions:** they cannot hold the job matrix of keys and platforms.
- **Calling the workflows at an SDK tag (0007):** a workflow-only fix needed a tag, and a tag also releases a host binary.
- **Calling them at `main`:** every engine build would pick up unreviewed workflow changes.

## Evidence

- `.github/workflows/engine-images.yml`
- `.github/workflows/engine-refresh.yml`
- `tools/images.sh`
- `tools/packages.sh`
- `tools/images_test.go`
