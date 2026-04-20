# Contributing to Tasks

Thank you for being so interested in helping develop Tasks. The time, skills, and perspectives you contribute to this project are valued.

## Issues and Proposals

Bugs, Proposals, & Feature Requests are all welcome. To get started, please open an issue via GitHub. Please provide as much detail as possible.

## Contributing

Contributions are always appreciated, please try to maintain usage contracts. If you are unsure, please open an issue to discuss.

## Local Verification

This repository includes a `Makefile` to keep local checks predictable:

- `make build` builds the package.
- `make tests` runs the test suite with the race detector enabled.
- `make benchmarks` runs the benchmark suite.
- `make coverage` writes coverage output to `coverage.out`.
- `make lint` runs `golangci-lint` when it is installed.
- `make format` applies `gofmt` and, when available, `golines`.

Please run `make tests` before opening a pull request. If you have `golangci-lint` installed locally, run `make lint` too.
