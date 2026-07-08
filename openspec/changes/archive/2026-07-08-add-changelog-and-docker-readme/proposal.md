## Why

The GHCR image and GoReleaser binary release pipeline already exist and match
what's specified under the pending `packaging-and-deployment` capability
(`openspec/changes/add-vbart-exporter/specs/packaging-and-deployment/spec.md`):
tag-only publishing, semver GHCR tags with the `v` prefix stripped, multi-arch
`linux/amd64`/`linux/arm64` images, and GoReleaser binaries attached to the
GitHub release. What's missing is that the README never tells a user the
image exists or how to run it from GHCR — it only documents building the
image locally. Separately, the project has no `CHANGELOG.md`, so there is no
record of what changed for the upcoming `v1.0.0` tag.

## What Changes

- Add a "README documents the published image" requirement to
  `packaging-and-deployment`: the README must link to the GHCR image
  (`ghcr.io/krom/vbart-meshcoretel-exporter`) and show a `docker run` example
  and a `docker-compose.yml` snippet that pull the published image instead of
  building locally.
- Update `README.md` accordingly (GHCR badge/link, `docker run` using the
  GHCR image, compose snippet using `image: ghcr.io/...`).
- Introduce a new `changelog-maintenance` capability: the repository
  maintains `CHANGELOG.md` following the Keep a Changelog format and
  Semantic Versioning, updated whenever a version is tagged.
- Add `CHANGELOG.md` with an `[Unreleased]` section and a `[1.0.0] -
  2026-07-08` entry describing the first release.

## Capabilities

### New Capabilities
- `changelog-maintenance`: repository maintains a `CHANGELOG.md` following
  Keep a Changelog + SemVer conventions, updated per tagged release.

### Modified Capabilities
- `packaging-and-deployment`: adds a requirement that the README documents
  and links the published GHCR image, with `docker run` and
  `docker-compose.yml` examples that pull it rather than building locally.

## Impact

- `README.md` — new Docker section content, GHCR link.
- New `CHANGELOG.md` at repo root.
- No code, workflow, or `.goreleaser.yaml` changes — CI/release automation
  already satisfies the existing `packaging-and-deployment` requirements.
- Note: `openspec/specs/` is currently empty because `add-vbart-exporter`
  (which first defined `packaging-and-deployment`) has not been archived
  yet. This change's delta spec for `packaging-and-deployment` should be
  read together with that pending change until both are archived/synced.
