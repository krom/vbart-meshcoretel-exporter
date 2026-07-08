## ADDED Requirements

### Requirement: README documents the published GHCR image
The `README.md` SHALL link to the published container image at
`ghcr.io/krom/vbart-meshcoretel-exporter` and SHALL include a `docker run`
example and a `docker-compose.yml` snippet that both reference the published
GHCR image (not only a locally-built image tag).

#### Scenario: User follows README without cloning the repo
- **WHEN** a user reads the README's Docker section
- **THEN** they find a `docker run` command that pulls and runs
  `ghcr.io/krom/vbart-meshcoretel-exporter` without needing to build the
  image themselves

#### Scenario: Compose example uses the published image
- **WHEN** a user reads the README's Docker Compose example
- **THEN** it shows an `image:` field referencing
  `ghcr.io/krom/vbart-meshcoretel-exporter` (in addition to any existing
  local-build instructions, which MAY remain for contributors)
