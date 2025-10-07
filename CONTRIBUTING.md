# Contributing to OTC

## Development Setup

```bash
    # Clone and install dependencies
    git clone https://github.com/mango-habanero/otc
    cd otc
    make deps
```

## Development Workflow

1. Create feature branch: `feat/OTC-X-description`
2. Write code
3. Run `make check` (runs fmt, lint, test)
4. Fix any issues
5. Commit using Angular conventional format
6. Push and create PR

## Pre-commit Checklist

Before every commit, run:
```bash
    make check
```

This runs:
- Code formatting check
- Linter
- All tests

No git hooks are configured. CI enforces these checks.

## Commit Message Format

Use Angular conventional commits:

```
<type>(<scope>): <subject>

<body>

Refs: OTC-X
```

**Types:** `feat`, `fix`, `docs`, `chore`, `test`, `refactor`, `ci`

**Example:**
```text
    feat(runtime): add containerd adapter
    
    - Implement CRI client wrapper
    - Add container lifecycle methods
    
    Refs: OTC-15
```

## Testing

- Unit tests: alongside code in `*_test.go`
- Integration tests: `test/integration/`
- E2E tests: `test/e2e/`

Use table-driven tests. See `pkg/otc/version_test.go` for example.

## Releases

Releases are automated via semantic versioning:
- Merges to `main` trigger automatic version detection
- Version determined by commit types (feat/fix/BREAKING CHANGE)
- Tags, changelog, and GitHub releases created automatically

No manual versioning needed.