# Project Architecture

## Directory Structure

- **pkg/otc/**: Public API - importable by external projects
- **internal/core/**: Private implementation details
- **api/proto/**: gRPC/protobuf definitions for runtime adapters
- **configs/**: Configuration templates
- **scripts/**: CI/CD and development scripts
- **test/**: Integration and E2E tests (unit tests live alongside code)

## Testing Strategy

- **Unit tests**: `*_test.go` files alongside source code
- **Integration tests**: `test/integration/` - external dependencies required
- **E2E tests**: `test/e2e/` - complete workflow validation
- **Pattern**: Table-driven tests (see `pkg/otc/version_test.go`)

## Development Workflow

See [CONTRIBUTING.md](CONTRIBUTING.md) for complete development guidelines.

## Design Decisions

### Why gRPC for Runtime Adapters?
Language-agnostic, efficient, strongly-typed communication between runtime adapters.

### Why No cmd/ Directory?
OTC is a library, not an application. CLI tooling may be added later if needed.