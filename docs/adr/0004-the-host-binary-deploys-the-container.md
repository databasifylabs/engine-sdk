# 0004. The host binary deploys the container

- Status: Accepted
- Date: 2026-11-10

## Context

The app used to deploy each node with its own Blade script. That script checked for Docker, pulled the pinned image, guarded against another database's data, created `/srv/<engine>`, ran the container with a fixed set of flags, replaced it when the image changed, and pruned old images. It also knew each engine's TLS file names. The app is being rebuilt so that it runs nothing on a database server except `databasify` commands, the same way for every engine.

## Decision

- `databasify host deploy {image, databaseId, nodeId}` runs in the host binary, outside any container. It:
  - pulls the image when missing;
  - reads the engine from the image's `io.databasify.engine` label;
  - refuses, before changing anything, an image without an `io.databasify.sdk` label, since an image not built on the SDK cannot answer the commands deploy relies on;
  - refuses with `busy`, before changing anything, when the `.database` marker or the running container's `io.databasify.database` label names another database, or when `data` holds files with no marker;
  - creates `/srv/<engine>`, `tls` and `conf`, owned by 64999, and writes the `.database` marker;
  - creates, starts or replaces `databasify-<engine>`;
  - prunes the repository's other images;
  - waits until the engine answers, then tunes the host with the engine's profile ([0006](0006-engines-declare-their-host-profile.md)).
- The run flags live in one place, `host/deploy.go`:
  - `--network host --ipc host --security-opt label=disable --restart unless-stopped --stop-timeout 960 --ulimit nofile=65536:65536`;
  - the `/srv/<engine>` volume;
  - the `io.databasify.database` and `io.databasify.node` labels;
  - the `DATABASIFY_IMAGE` environment variable.
- `databasify host remove {databaseId, erase}` stops the node. With `erase`, it restores the host's tuning and removes only the directory whose `.database` marker matches.
- `databasify tls install {certificate, key, ca}` writes the files each engine reads, so the app never names them.
- The app installs the official host binary on its own, and `host deploy` leaves that binary alone ([0005](0005-images-are-the-only-release.md)).

## Consequences

- Upgrades, redeploys and new nodes all go through `host deploy`. A patch upgrade is `host deploy` with a new digest.
- The stop timeout is 960 seconds for every engine, enough for the slowest engine to stop. Engines that stop faster finish sooner.
- A server's data can only be reused by the database that wrote it, or after `host remove` with `erase`.

## Alternatives considered

- **Keep the deploy script in the app:** every engine detail it carries (TLS file names, data guard, run flags) would stay in the app, against the rule that the app only calls `databasify`.
- **A container command for deploy:** a container can't create or replace itself, so this has to run on the host.

## Evidence

- `host/deploy.go`
- `host/commands.go`
- `nodefs/tls.go`
