# Contributing

Contributions are welcome via pull request.

Before submitting:

```sh
go build ./...
go vet ./...
golangci-lint run ./...
go test -race ./...
```

If you change the metric mapping in `internal/metrics`, update the metric
reference table in `README.md` and the Grafana dashboard in
`dashboards/vbart-meshcoretel-exporter.json` to match.

By contributing, you agree that your contributions will be licensed under
the project's GPL-3.0-only license (see `LICENSE`).
