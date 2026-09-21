## Summary

<!-- What does this change do, and why? -->

## Related issue

<!-- e.g. Closes #123 -->

## How was this tested?

<!-- Commands run, new tests added, manual verification -->

## Checklist

- [ ] `gofmt -l .` is clean (`make fmt`)
- [ ] `go vet ./...` passes
- [ ] `go build ./...` succeeds
- [ ] `go test ./...` passes, and new behavior has table-driven tests
- [ ] New generation stages respect the budget/caps (no silent truncation)
- [ ] This change keeps pwprofiler an authorized-testing tool (see CONTRIBUTING.md)
