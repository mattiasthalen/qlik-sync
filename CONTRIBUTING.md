# Contributing to qs

Thanks for your interest in contributing!

## Prerequisites

- [Go](https://go.dev/dl/) 1.23+
- [qlik-cli](https://qlik.dev/toolkits/qlik-cli/)
- [golangci-lint](https://golangci-lint.run/welcome/install/) v2
- [Lefthook](https://github.com/evilmartians/lefthook)

## Development Setup

```bash
git clone https://github.com/mattiasthalen/qlik-sync.git
cd qlik-sync
lefthook install
make build
make test
```

## Testing

```bash
# Unit tests with race detector
make test

# Linting
make lint

# Static analysis
make vet

# Integration test (uses mock qlik-cli)
go test ./test/integration/ -v

# Coverage report
make coverage
```

## Workflow

1. Work in **git worktrees** on feature branches
2. Open a **draft PR** when starting work
3. Follow **TDD**: write a failing test, implement, verify it passes, commit
4. Use **conventional commits** with scope (e.g., `feat(sync): add space filter`)
5. Push after every commit
6. Use **merge commits** when merging (preserve full history)

## Pre-commit Hooks

Lefthook runs automatically on commit:

- `go vet` — static analysis
- `golangci-lint` — linting
- `go test` — full test suite
- Conventional commit message validation

## CI

Every PR runs vet, lint, and test. All checks must pass before merge.
