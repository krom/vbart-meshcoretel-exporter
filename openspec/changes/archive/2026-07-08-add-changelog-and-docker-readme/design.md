## Context

CI/release automation already exists and works (`.github/workflows/docker.yml`,
`release.yml`, `.goreleaser.yaml`). This change only touches documentation
(`README.md`) and adds a new tracked file (`CHANGELOG.md`); no code or
workflow changes are needed.

## Goals / Non-Goals

**Goals:**
- README tells users the image is published to GHCR and how to run it from
  there, without removing the existing local-build instructions.
- Establish a durable, low-friction changelog convention (Keep a Changelog +
  SemVer) and seed it with the `1.0.0` entry.

**Non-Goals:**
- No changes to `docker.yml`, `release.yml`, or `.goreleaser.yaml` — their
  tagging/publishing behavior already matches the spec.
- No changelog automation/tooling (e.g. auto-generation from commits) — this
  change only establishes the manual convention and file format.

## Decisions

- **README structure**: keep "Binary" and "Docker (build locally)"
  subsections as-is, add a new "Docker (GHCR image)" subsection above them
  showing `docker pull`/`docker run` against
  `ghcr.io/krom/vbart-meshcoretel-exporter:latest`, and add an `image:` line
  variant of the compose snippet. Rationale: don't delete instructions
  useful for contributors building from source; additive is lower risk.
- **CHANGELOG format**: Keep a Changelog (`## [Unreleased]` +
  `## [X.Y.Z] - YYYY-MM-DD` sections, `Added`/`Changed`/`Fixed`/etc.
  subheadings) since it's the de facto standard and pairs naturally with the
  existing SemVer git-tag convention. `1.0.0` entry dated 2026-07-08 under
  `Added`, describing the initial release.

## Risks / Trade-offs

- [README GHCR link goes stale if the repo is renamed/moved] → link uses the
  same `github.repository`-derived path already used in `docker.yml`
  (`krom/vbart-meshcoretel-exporter`), consistent with the rest of the repo.
- [CHANGELOG drifts from reality if not updated per release] → out of scope
  to enforce here; documented as a maintenance convention only.
