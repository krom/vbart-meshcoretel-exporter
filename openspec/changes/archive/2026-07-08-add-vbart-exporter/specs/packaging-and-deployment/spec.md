## ADDED Requirements

### Requirement: Multi-stage Docker image
The repository SHALL contain a multi-stage `Dockerfile`: a Go builder stage (`CGO_ENABLED=0`, unit tests executed during build) and a minimal final stage (distroless static or scratch with CA certificates) containing only the binary, running as a non-root user, with `EXPOSE 9642` and the exporter as entrypoint.

#### Scenario: Image build and run
- **WHEN** `docker build .` is executed and the resulting image is run without arguments
- **THEN** the build succeeds, the container serves `GET /-/healthy` with HTTP 200, and the container user is not root

#### Scenario: Test failure blocks image
- **WHEN** a unit test fails
- **THEN** `docker build` fails

### Requirement: Docker Compose examples
The repository SHALL contain a `docker-compose.yml` running the exporter, and an example compose stack (under `examples/`) adding Prometheus with a working scrape config for at least one device target and Grafana provisioned with the shipped dashboard and Prometheus datasource.

#### Scenario: Demo stack
- **WHEN** the example compose stack is started with a reachable device configured
- **THEN** Prometheus scrapes the exporter and the Grafana dashboard displays device data without manual setup

### Requirement: Continuous integration
GitHub Actions workflows SHALL: on every push and pull request run `go vet`, `golangci-lint`, and `go test -race ./...`; only on version tags (`v*.*.*`, semantic versioning) build and publish multi-arch (linux/amd64, linux/arm64) images to `ghcr.io` and produce release binaries via GoReleaser. Docker builds/publishes SHALL NOT run on plain branch pushes (including the default branch) — only on tag pushes.

#### Scenario: Pull request checks
- **WHEN** a pull request is opened
- **THEN** lint and tests run and failures block the merge

#### Scenario: Tagged release
- **WHEN** a tag `v1.0.0` is pushed
- **THEN** a GitHub release with binaries is published, and a `ghcr.io` image is published tagged `1.0.0`, `1.0`, `1`, and `latest` (the `ghcr.io` tags never carry the `v` prefix used in the git tag)

#### Scenario: Non-tag push does not publish an image
- **WHEN** a commit is pushed to `main` or any other branch without an accompanying version tag
- **THEN** no `ghcr.io` image is built or published

### Requirement: GPL-3.0 licensing
The repository SHALL be licensed GPL-3.0-only with the full license text in `LICENSE`, a license notice in the README, and license/copyright headers where conventional. Dependencies SHALL be limited to GPLv3-compatible licenses (Apache-2.0, BSD, MIT); adding a GPLv3-incompatible dependency is prohibited.

#### Scenario: License presence
- **WHEN** the repository is published
- **THEN** `LICENSE` contains the GPL-3.0 text and the Go module's dependency licenses are all GPLv3-compatible
