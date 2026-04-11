# Contributing to xk6-valkey

Thank you for your interest in contributing to xk6-valkey! This document provides guidelines and instructions for contributing.

## Getting Started

### Prerequisites

- [Go toolchain](https://go101.org/article/go-toolchain.html) (see `go.mod` for the required version)
- [xk6](https://github.com/grafana/xk6) (`go install go.k6.io/xk6/cmd/xk6@latest`)
- [golangci-lint](https://golangci-lint.run/welcome/install/)
- Git

### Setup

1. Fork the repository
2. Clone your fork:
   ```shell
   git clone https://github.com/<your-username>/xk6-valkey.git
   cd xk6-valkey
   ```
3. Install dependencies:
   ```shell
   go mod download
   ```

## Development Workflow

### Building

```shell
make build
```

### Running Tests

```shell
make test
```

Tests use an in-process RESP protocol stub server over TCP, so no running Valkey instance is required.

### Running Linters

```shell
make lint
```

The linter configuration is downloaded from the main k6 repository on first run.

### Full Check

```shell
make check
```

This runs both linters and tests.

## Submitting Changes

1. Create a feature branch from `main`:
   ```shell
   git checkout -b feature/your-feature-name
   ```
2. Make your changes
3. Ensure all tests pass and linters are clean:
   ```shell
   make check
   ```
4. Commit your changes with a clear commit message
5. Push your branch and open a Pull Request against `main`

### Pull Request Guidelines

- Keep changes focused — one feature or fix per PR
- Add tests for new functionality
- Update documentation if the public API changes
- Ensure CI passes before requesting review

## Reporting Issues

- Use [GitHub Issues](https://github.com/moko-poi/xk6-valkey/issues) to report bugs or request features
- Include steps to reproduce for bug reports
- Check existing issues before creating a new one

## License

By contributing, you agree that your contributions will be licensed under the [Apache License 2.0](LICENSE).
