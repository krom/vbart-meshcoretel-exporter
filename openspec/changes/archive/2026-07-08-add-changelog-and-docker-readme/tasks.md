## 1. README: document the published GHCR image

- [x] 1.1 Add a "Docker (GHCR image)" subsection to the README's Quick
      Start with a `docker pull`/`docker run` example against
      `ghcr.io/krom/vbart-meshcoretel-exporter:latest`
- [x] 1.2 Add an `image:`-based `docker-compose.yml` snippet using the
      GHCR image, alongside (not replacing) the existing local-build compose
      example
- [x] 1.3 Link to the GHCR package page from the README (e.g. near the
      existing CI/Docker badges)

## 2. CHANGELOG.md

- [x] 2.1 Create `CHANGELOG.md` at repo root with Keep a Changelog header,
      an empty `## [Unreleased]` section, and a `## [1.0.0] - 2026-07-08`
      section under `Added` describing the first release
- [x] 2.2 Cross-check the `1.0.0` entry wording against actual shipped
      functionality (scrape handler, metrics, Docker/Compose examples,
      dashboards) so it isn't just a placeholder

## 3. Verification

- [x] 3.1 Re-read README Docker section end-to-end for accuracy and
      consistency with existing sections
- [x] 3.2 Confirm `CHANGELOG.md` renders correctly as Markdown and version
      heading matches the `v1.0.0` tag convention used by
      `.github/workflows/docker.yml` / `release.yml`
