# Architecture Decision Records

An ADR records one decision that shapes the SDK beyond a single change: what was decided, why, and what it costs.

## Rules

- One decision per file, named `NNNN-kebab-slug.md`, numbered in order.
- Status is one of `Proposed`, `Accepted`, `Superseded by NNNN` or `Deprecated`.
- An accepted decision is not rewritten. To change a decision, write a new ADR and mark the old one `Superseded by NNNN`.
- Facts in an ADR (names, numbers, paths, behaviour) are kept in sync with the code in the same change that alters them.
- Write an ADR only for a decision that outlives the change that introduced it.
- Evidence cites repository paths only.

## Template

```markdown
# NNNN. Title

- Status: Proposed
- Date: YYYY-MM-DD

## Context

What forces are at play and what problem needs a decision.

## Decision

What we do, stated plainly.

## Consequences

What becomes easier, what becomes harder, and which constraints follow.

## Alternatives considered

Options that were tried or rejected, and why.

## Evidence

- `path/to/file`
```

## Index

| # | Decision | Status |
| --- | --- | --- |
| [0001](0001-the-public-engine-sdk.md) | The public engine SDK | Accepted |
| [0002](0002-typed-commands-are-the-engine-interface.md) | Typed `databasify` commands are the only engine interface | Accepted |
| [0003](0003-hand-written-go-contract.md) | The contract is hand-written Go in Laravel-style packages | Accepted |
| [0004](0004-the-host-binary-deploys-the-container.md) | The host binary deploys the container | Accepted |
| [0005](0005-images-are-the-only-release.md) | Images are the only release | Accepted |
| [0006](0006-engines-declare-their-host-profile.md) | Engines declare their host profile | Accepted |
| [0007](0007-engines-publish-through-the-sdk-workflows.md) | Engines publish through the SDK's reusable workflows | Superseded by 0008 |
| [0008](0008-engines-call-the-sdk-workflows-at-a-pinned-commit.md) | Engines call the SDK workflows at a pinned commit | Accepted |

ADRs 0002 to 0005 were carried over from `databasifylabs/engines` (there 0001, 0002, 0009 and 0013), with paths and facts updated for this repository.
