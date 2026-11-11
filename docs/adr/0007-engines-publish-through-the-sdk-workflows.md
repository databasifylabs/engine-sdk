# 0007. Engines publish through the SDK's reusable workflows

- Status: Superseded by 0008
- Date: 2026-11-11

## Context

Every engine, the team's and a plugin author's, lives in its own repository and publishes images the same way: check the pins, hash what an image is built from, build and smoke-test both platforms, scan, publish, and refresh the pins weekly. Copying that CI into each repository lets the copies drift, and an author would maintain release logic that is not theirs.

## Decision

- The SDK ships `engine-images.yml` and `engine-refresh.yml` as reusable workflows with the inputs `engine` and `versions-file`. An engine repository calls them at the SDK version its `go.mod` requires; it holds `images/<engine>/` and a `versions.json` with `registry`, `sdk`, `go` and its own lines.
- `tools/images.sh` runs from the engine repository root, reads that repository's `versions.json` and `images/`, and takes its helper scripts from the SDK checkout the workflow makes at the `go.mod` version.
- An image's content hash covers the engine's Go sources, `go.mod` and `go.sum` (so the SDK version), its image files and its pins. Every `replace` to a local path (the SDK, or a sibling engine such as `engine-mysql` for MariaDB) must be written `../<dir>`; its Go sources join the hash and are staged into the build context `modules`, which every build passes (empty without replacements) and every engine Dockerfile copies to `/src/`. A missing replacement fails the hash. Dropping the replacements once the modules are tagged needs no Dockerfile change.
- `engine-images.yml` runs the engine's `go vet` and `go test -race` before it builds, so an engine repository needs no Go workflow of its own.
- `check` refuses a `versions.json` `sdk` that differs from the version `go.mod` requires.
- `<release>`, `<line>` and `latest` move only on the caller's default branch; every other branch publishes `<release>-<sha7>` only.

## Consequences

- A fix to the release logic reaches an engine when it moves to the SDK release carrying it, like any other SDK change.
- Engine repositories keep two short caller workflows; the build and publish logic has one home.
- The caller's images workflow must be named `images.yml`: the refresh dispatches it on the pull request branch.
- Only tagged SDK versions can be checked out for the tools; a pseudo-version in `go.mod` does not resolve.

## Alternatives considered

- **A template that copies the workflows into each engine:** the copies drift and every fix is a pull request per engine.
- **Composite actions:** they cannot hold the job matrix of keys and platforms.

## Evidence

- `.github/workflows/engine-images.yml`
- `.github/workflows/engine-refresh.yml`
- `tools/images.sh`
- `tools/packages.sh`
- `tools/images_test.go`
