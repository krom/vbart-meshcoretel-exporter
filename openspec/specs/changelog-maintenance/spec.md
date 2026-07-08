# changelog-maintenance Specification

## Purpose

Defines how the repository maintains `CHANGELOG.md`: format conventions and
the initial release entry.

## Requirements

### Requirement: CHANGELOG.md follows Keep a Changelog and SemVer
The repository SHALL contain a `CHANGELOG.md` at the repo root formatted
per [Keep a Changelog](https://keepachangelog.com/en/1.1.0/): an
`## [Unreleased]` section at the top, followed by dated version sections
`## [X.Y.Z] - YYYY-MM-DD` in descending order, using standard subheadings
(`Added`, `Changed`, `Fixed`, `Removed`, etc.) grouping entries. Version
numbers SHALL follow [Semantic Versioning](https://semver.org/) and match
the corresponding `vX.Y.Z` git tag (without the `v` prefix).

#### Scenario: Repository contains a changelog
- **WHEN** the repository is inspected at any commit after this change
- **THEN** `CHANGELOG.md` exists at the repo root with an `[Unreleased]`
  section and at least one dated version section

#### Scenario: Version entry matches the git tag
- **WHEN** a version section `## [1.0.0] - 2026-07-08` exists in
  `CHANGELOG.md`
- **THEN** a corresponding `v1.0.0` git tag exists (or is created) for that
  release, with the `v` prefix stripped in the changelog heading

### Requirement: Initial 1.0.0 changelog entry
`CHANGELOG.md` SHALL include a `## [1.0.0] - 2026-07-08` section
documenting the first release of the exporter.

#### Scenario: First release documented
- **WHEN** a reader opens `CHANGELOG.md` looking for the first release
- **THEN** they find a `[1.0.0] - 2026-07-08` section with an `Added`
  subsection describing the initial release of
  `vbart-meshcoretel-exporter`
