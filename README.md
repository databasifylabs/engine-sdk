# Databasify Engine SDK

- [Introduction](#introduction)
- [Using the Runtime in an Image](#using-the-runtime-in-an-image)
- [Calling Commands](#calling-commands)
- [Runtime](#runtime)
- [Host Binary](#host-binary)
- [Building Images](#building-images)
- [Publishing Images](#publishing-images)
- [Versioning](#versioning)
- [Testing](#testing)

## Introduction

The engine SDK is the generic `databasify` binary that every Databasify engine image runs. It holds:
- the process supervisor, PID 1 in every engine container;
- the runtime commands Databasify uses to run its scripts, files, tasks and services inside the container;
- the host binary that deploys and removes the engine container;
- the runtime conformance suite;
- the tools and workflows that build and publish engine images.

Nothing in it names a specific engine. Engine behaviour lives in Databasify's PHP drivers and Blade scripts. An engine image is the official database image, its tools and this binary.

## Using the Runtime in an Image

Copy the binary from the published SDK image, pinned by digest, into the engine image:

```dockerfile
ARG BASE_IMAGE
FROM ghcr.io/databasifylabs/engine-sdk:1.0.0@sha256:<digest> AS sdk

FROM ${BASE_IMAGE}
ARG SDK_VERSION
COPY --from=sdk /databasify /usr/local/bin/databasify
LABEL io.databasify.engine="example" \
      io.databasify.sdk="${SDK_VERSION}"
ENTRYPOINT ["/usr/local/bin/databasify", "serve"]
```

- **`io.databasify.engine`** names the engine. It is the container name (`databasify-<engine>`), the data root (`/srv/<engine>`) and `DATABASIFY_ENGINE`.
- **`io.databasify.sdk`** is the SDK version. `tools/images.sh args` passes it as `SDK_VERSION`, from the `sdk` pin in `versions.json`.
- **`conf/processes.json`** lists the processes the supervisor runs. Databasify writes it on every `node configure`.

A new engine starts from [engine-template](https://github.com/databasifylabs/engine-template).

## Calling Commands

Every command reads one JSON object on stdin and writes one on stdout.
- Quick commands reply at once.
- Tasks reply with their operation, which the caller polls with `task status`.
- Errors reply with a `code`:
  - exit `64` for `invalid_request`;
  - exit `70` for `internal`;
  - exit `1` for every other code: `unsupported`, `not_found`, `already_exists`, `not_primary`, `busy` and `unavailable`.

A well-formed command the runtime does not serve, such as an engine command from before 1.0, answers `unsupported`.

## Runtime

| Command | Request | Reply |
|---|---|---|
| `engine info` | `{}` | `engine`, `line`, `version`, `sdk` |
| `runtime info` | `{}` | `protocol` (`1`), `engine`, `sdk` |
| `runtime run` | `script`, `env`, `user`, `timeoutSeconds` (1–3600) | `exit`, `stdout`, `stderr`, `truncated` |
| `task start` | `key`, `script`, `env`, `user` | the task's operation |
| `task status`, `task stop` | `key` | the task's operation |
| `task log` | `key` | the last 50 `lines` of the task's log |
| `file put` | `path`, `contentBase64`, `mode`, `owner` | `{}` |
| `file get` | `path` | `contentBase64`, `exists` |
| `service start`, `stop`, `restart`, `status` | `units` | `{}`, or each unit's `activeState`, `subState` and `result` |
| `service reload` | `{}` | `{}`, after re-reading `conf/processes.json` |
| `logs read` | `unit`, `lines` (1–10000) | the last log lines |
| `logs forward` | `destination` | `{}` |

- **Scripts run as `bash -euo pipefail -s`, with the script on stdin and never in argv.** Passwords inside a script never show up in `ps`.
- **`runtime run` limits:**
  - stdout and stderr are each capped at 1 MiB, and `truncated` says when output was cut;
  - a timeout kills the whole process group and replies with exit `124`.
- **Tasks are operations with a caller-chosen key** (`^[a-z0-9][a-z0-9.-]{0,79}$`):
  - starting a key again returns the existing task instead of running it twice;
  - a non-zero exit fails the task with `script exited <code>: <last stderr line>`.
- **File paths are relative to the engine home.** Absolute paths, `..`, and paths that leave the home through a symlink are refused with `invalid_request`. Writes are atomic.
- **Inside the container, scripts control units with `databasify ctl`.**

```bash
echo '{"script":"psql -XAtc \"SELECT 1\"","env":{},"user":"postgres","timeoutSeconds":30}' | databasify runtime run
# {"exit":0,"stdout":"1\n","stderr":"","truncated":false}
```

## Host Binary

Databasify installs the same binary on every server, copied from the SDK image. Outside a container it runs:

- **`host deploy`:** runs the engine container from the database's pinned image digest, and applies the `hostProfile` in the request (sysctl settings, floors and transparent hugepages). The image is never asked for a profile.
- **`host remove`:** stops and removes the container, and erases its data when asked.
- **`host tune`:** applies a `hostProfile` again.
- **`host restore`:** restores the container's data root.

It forwards every other well-formed `<group> <verb>` into the engine container.

## Building Images

An engine repository holds `images/<engine>/` (the `Dockerfile`, `smoke/` and an optional `conformance` marker) and a `versions.json` with the `sdk` pin and one entry per version line. Run `tools/images.sh` from the engine repository root:

```bash
$IMAGES check
$IMAGES build <engine>-<line> --load --tag smoke:<engine>-<line>
$IMAGES smoke smoke:<engine>-<line> <engine>
$IMAGES refresh <engine>-<line>
```

- `$IMAGES` is `tools/images.sh` in a clone of this repository at `v<sdk>`, the tag the `sdk` pin names.
- `check` refuses an `sdk` pin that is not a 1.x release.
- With the `conformance` marker, `smoke` runs this repository's runtime conformance suite against the smoke container.

## Publishing Images

An engine repository calls two reusable workflows at a full SDK commit SHA (see [ADR 0008](docs/adr/0008-engines-call-the-sdk-workflows-at-a-pinned-commit.md)):

```yaml
jobs:
  images:
    uses: databasifylabs/engine-sdk/.github/workflows/engine-images.yml@<sha>
    with:
      engine: example
      versions-file: versions.json
```

`engine-images.yml` works in this order:
1. It reads the `sdk` pin and checks out the SDK tools at that tag.
2. It builds, smoke-tests and scans every changed line for `linux/amd64` and `linux/arm64`.
3. It publishes `<registry>/databasify-<engine>:<release>-<sha7>`. On the default branch it also moves `<release>` and `<line>`, and `latest` for the newest line.

`engine-refresh.yml` opens a pull request every week with the newest pins inside each line.

`registry` in `versions.json` must be `ghcr.io/<owner of the engine repository>`. A new package starts private. Make it public, because Databasify pulls images without credentials.

## Versioning

`1.0` is the final runtime ([ADR 0009](docs/adr/0009-engine-sdk-1-0-is-a-frozen-runtime.md)).
- After `1.0.0`, releases are patches only: bug and security fixes, with no new commands or fields.
- A new engine or engine feature never changes the SDK. It ships as a Databasify driver and Blade scripts.

Every image carries the SDK version in its `io.databasify.sdk` label, and `engine info` reports it. Databasify 0.2 accepts `^0.2.2 || ^1.0`.

## Testing

```bash
go vet ./...
go test -race -count=1 ./...
```

`conformance.Run(t, conformance.DockerRunner{Container: name})` runs the runtime conformance suite against a running engine container. It checks:
- `engine info` and `runtime info`;
- `runtime run`, including that secrets passed in `env` never reach the process list;
- a file round trip and a task;
- that invalid input is refused and engine commands answer `unsupported`.

Decisions are recorded in [docs/adr](docs/adr/README.md).
