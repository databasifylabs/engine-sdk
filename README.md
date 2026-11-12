# Databasify Engine SDK

- [Introduction](#introduction)
- [Installation](#installation)
- [Writing an Engine](#writing-an-engine)
- [Command Groups](#command-groups)
- [Calling Commands](#calling-commands)
- [Runtime](#runtime)
- [Flows](#flows)
- [Building Images](#building-images)
- [Publishing Images](#publishing-images)
- [Versioning](#versioning)
- [Testing](#testing)

## Introduction

The engine SDK is the Go framework every Databasify database engine is built on. It holds:
- the engine contract and the typed command router;
- the process supervisor;
- the host binary;
- the conformance suite;
- the tools and workflows that build and publish engine images.

Nothing in it names a specific engine.

## Installation

Add the SDK to an engine module. A new engine should start from [engine-template](https://github.com/databasifylabs/engine-template).

```bash
go get github.com/databasifylabs/engine-sdk@v0.2.1
```

## Writing an Engine

An engine implements `contract.Engine`, and its `main` is one call:

```go
package main

import (
	"os"

	sdk "github.com/databasifylabs/engine-sdk"
	"example.com/engine-example/engine"
)

func main() {
	os.Exit(sdk.Run(engine.New()))
}
```

`Name()` is the engine id. It is also the image's `io.databasify.engine` label and `DATABASIFY_ENGINE`.

`Capabilities()` lists the ports, the reserved users and databases, and the password encryptions. `HostProfile()` holds the sysctl settings and floors that `host tune` applies, and whether to turn off transparent hugepages.

## Command Groups

`contract.Engine` has one accessor per group: `Nodes`, `Cluster`, `Accounts`, `Archive`, `Migration`, `Pools` and `Config`.

- An engine offers a whole group unless it returns that group's `Unsupported*` stand-in. The stand-in's commands answer `unsupported` and stay out of `engine capabilities`.
- A group that implements only some of its commands implements `contract.Partial`:

```go
func (Nodes) Implements() []string {
	return []string{"node status", "node configure"}
}
```

## Calling Commands

Every command reads one JSON object on stdin and writes one on stdout.
- Quick commands reply at once.
- Long commands start an operation and reply with its key.
- Errors reply with a `code`:
  - exit `64` for `invalid_request`;
  - exit `70` for `internal`;
  - exit `1` for every other code: `unsupported`, `not_found`, `already_exists`, `not_primary`, `busy` and `unavailable`.

```bash
echo '{}' | databasify node status
# {"role":"primary","version":"18.6"}
```

`cmd/databasify-host` is the host binary that Databasify installs on every server. It runs `host deploy`, `host remove`, `host tune` and `host restore`, and forwards every other well-formed `<group> <verb>` into the engine container.

## Runtime

Every engine answers these engine-neutral commands. The app uses them to run its own scripts inside the container.

| Command | Request | Reply |
|---|---|---|
| `runtime info` | `{}` | `protocol` (`1`), `engine`, `sdk` |
| `runtime run` | `script`, `env`, `user`, `timeoutSeconds` (1–3600) | `exit`, `stdout`, `stderr`, `truncated` |
| `task start` | `key`, `script`, `env`, `user` | the task's operation |
| `task status`, `task stop` | `key` | the task's operation |
| `task log` | `key` | the last 50 `lines` of the task's log |
| `file put` | `path`, `contentBase64`, `mode`, `owner` | `{}` |
| `file get` | `path` | `contentBase64`, `exists` |
| `service start`, `stop`, `restart`, `status` | `units` | `{}`, or each unit's `activeState`, `subState` and `result` |
| `service reload` | `{}` | `{}`, after re-reading `conf/processes.json` |

- **Scripts run as `bash -euo pipefail -s`, with the script on stdin and never in argv.** Passwords inside a script never show up in `ps`.
- **`runtime run` limits:**
  - stdout and stderr are each capped at 1 MiB, and `truncated` says when output was cut;
  - a timeout kills the whole process group and replies with exit `124`.
- **Tasks are operations with a caller-chosen key** (`^[a-z0-9][a-z0-9.-]{0,79}$`):
  - starting a key again returns the existing task instead of running it twice;
  - a non-zero exit fails the task with `script exited <code>: <last stderr line>`.
- **File paths are relative to the engine home.** Absolute paths, `..`, and paths that leave the home through a symlink are refused with `invalid_request`. Writes are atomic.

```bash
echo '{"script":"psql -XAtc \"SELECT 1\"","env":{},"user":"postgres","timeoutSeconds":30}' | databasify runtime run
# {"exit":0,"stdout":"1\n","stderr":"","truncated":false}
```

## Flows

Each flow is the same on every engine. Run each step on the node it names; "every node" means every node still in the database. Steps marked *long* return an operation that Databasify polls with `operation status`.

**New node**, before anything else on a new server:

1. `host deploy` with the database's pinned digest.
2. `tls install` with the node's certificate.

**Create** a single node or a cluster:

1. Every node: new node, then `node configure`, `node start`, `node ready` (*long*).
2. Every node: `node bootstrap`. It creates the users on the seed and does nothing elsewhere.
3. Every clustered node: `cluster enable` (*long*), in any order. Joiners wait for the seed.

**Add a node:**

1. The new node: new node, then `node configure` with the new member list, `node start`, `node ready` (*long*).
2. Every existing node: `node configure` with the new member list, then `cluster membership`.
3. The new node: `cluster enable` (*long*). It copies the data and joins.

**Remove a node:**

1. If the leaving node is the primary: `cluster failover` to a standby.
2. Take the leaving node out of DNS and the connection endpoints.
3. The leaving node: `cluster disable`.
4. Every remaining node: `node configure` without the leaving node, then `member forget` with its IP, then `cluster membership`.
5. The leaving node: `host remove` with `erase`.

**Planned failover:** `cluster failover` with the target standby, once, on any voting node. Read-only nodes follow the new primary on their own.

**Recovery after an outage**, when the cluster has lost its majority:

1. Every reachable voting node: `node wake`, then `cluster position`.
2. The node with the highest position: `cluster force-quorum` with `othersDown`, only once the other voting nodes are confirmed down.
3. Each node that comes back: `cluster rejoin` (*long*).

**Rejoin** a node that was down or fell behind: `cluster rejoin` (*long*) on that node. It may copy the whole dataset.

**Patch upgrade**, one node at a time: read-only nodes, then standbys, then `cluster failover` to an upgraded standby, then the former primary. For each node:

1. `host deploy` with the new digest. The engine's processes come back on their own.
2. `node configure`, so the new image writes its config files.
3. `node ready` (*long*).
4. On a cluster member: `cluster rejoin` (*long*).

**Settings change:**

1. Every node: `node configure` with the new `settings`.
2. Any one node: `config sync`.
3. Every node: `config apply`.
4. If `config show` reports `pendingRestart`, restart one node at a time: read-only nodes, then standbys, then `cluster failover` to a restarted standby, then the former primary. For each node: `node restart`, `node ready` (*long*), and on a cluster member `cluster rejoin` (*long*).

**Re-tune after a resize:**

1. Every node: `node configure` with the new sizes in `members`.
2. Every node: `config tune`, then `config apply`.
3. Restart one node at a time, as in a settings change, if `config tune` lists settings in `restart`.

**Restore to a point in time:**

1. A new single node: new node, then `node configure` with its own `archive`, `node start`, `node ready` (*long*), `node bootstrap`.
2. The same node: `restore start` (*long*) with the source's storage and the target time.
3. Grow it into a cluster with **add a node**.

**Rotate certificates:** every node, `tls install` with its new certificate.

**Delete:** every node, `host remove` with `erase`.

## Building Images

An engine repository holds `images/<engine>/` (the `Dockerfile`, `smoke/` and an optional `conformance` marker) and a `versions.json` with one entry per version line. Run `tools/images.sh` from the engine repository root:

```bash
$IMAGES check
$IMAGES build <engine>-<line> --load --tag smoke:<engine>-<line>
$IMAGES smoke smoke:<engine>-<line> <engine>
$IMAGES refresh <engine>-<line>
```

`$IMAGES` is `tools/images.sh` in a clone of this repository, at the version the engine's `go.mod` requires.

To build against a local checkout of the SDK or of another engine module, add a `replace` that points at a sibling directory, written `../<dir>`. The tools copy its Go sources into the `modules` build context, and every engine Dockerfile starts its build stage with `COPY --from=modules . /src/`.

## Publishing Images

An engine repository calls two reusable workflows at a full SDK commit SHA (see [ADR 0008](docs/adr/0008-engines-call-the-sdk-workflows-at-a-pinned-commit.md)):

```yaml
jobs:
  images:
    uses: databasifylabs/engine-sdk/.github/workflows/engine-images.yml@398de5e1c8023525ca7b1990e2b39525017b2c4f
    with:
      engine: example
      versions-file: versions.json
```

`engine-images.yml` works in this order:
1. It runs `go vet` and the race tests.
2. It builds, smoke-tests and scans every changed line for `linux/amd64` and `linux/arm64`.
3. It publishes `<registry>/databasify-<engine>:<release>-<sha7>`. On the default branch it also moves `<release>` and `<line>`, and `latest` for the newest line.

`engine-refresh.yml` opens a pull request every week with the newest pins inside each line.

`registry` in `versions.json` must be `ghcr.io/<owner of the engine repository>`. A new package starts private; make it public, because Databasify pulls images without credentials.

## Versioning

The SDK follows semantic versioning.
- Within a major version, releases only add: new commands, new optional fields and new reply fields. An engine that lacks a newer command answers `unsupported`.
- A breaking change is a new major version, with an upgrade guide.
- Before `1.0`, any minor release may break.

Every image carries the SDK version in its `io.databasify.sdk` label, and `engine info` reports it. Each Databasify release runs one SDK range:

| Databasify | SDK range |
| --- | --- |
| Next release | `^0.1` |

## Testing

```bash
go vet ./...
go test -race -count=1 ./...
```

`conformance.Run(t, conformance.DockerRunner{Container: name})` runs the conformance suite against a running engine container. The suite always tests `engine capabilities`, `engine host-profile`, `engine info` and `node status`. It tests every other command once the engine lists it, and checks that every unlisted command answers `unsupported`.

Decisions are recorded in [docs/adr](docs/adr/README.md).
