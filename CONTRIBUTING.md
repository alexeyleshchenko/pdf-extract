# Contributing

- Run tests from the repo root: `go test ./... -count=1`
- Poppler tests need `pdfinfo` and `pdftotext` on `PATH` (`poppler-utils` on Debian/Ubuntu). Without them, those tests skip locally and fail in CI.
- Format: `gofmt -w .` (or your editor’s Go format on save)
- Open pull requests against `main`; describe the change and any deploy or config impact.

CI installs `poppler-utils`, runs tests, and builds the Docker image on relevant pushes to `main`.
