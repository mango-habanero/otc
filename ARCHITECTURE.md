# Project Architecture

This project follows a modular Go structure inspired by community best practices.

- **cmd/**: Application entry points. Each subfolder produces a binary (e.g., `cmd/otc`).
- **pkg/**: Publicly reusable Go packages (importable by other projects).
- **internal/**: Private packages (only for use inside this module).
- **api/**: API definitions (OpenAPI, gRPC, GraphQL).
- **configs/**: Configuration file templates (YAML, JSON, TOML).
- **scripts/**: Utility scripts for CI/CD and developer workflows.
- **test/**: Integration and end-to-end tests.
