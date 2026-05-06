# Developing

## Getting started

In order to test and develop in this repo you will need the following dependencies installed:
- make

After cloning the repo run `make bootstrap` to download go mod dependencies, create the `/.tmp` dir, and download helper utilities.

The main `make` tasks for common static analysis and testing are `lint`, `lint-fix`, and `unit`.

See `make help` for all the current make tasks.

## Multi-module layout

This repository contains three Go modules:

- `github.com/anchore/go-logger` — root module (interface, `discard`, `redact`, `slog`)
- `github.com/anchore/go-logger/adapter/logrus` — own module
- `github.com/anchore/go-logger/adapter/charm` — own module

Each adapter sub-module declares a `replace github.com/anchore/go-logger => ../..` directive so that local development against the parent module Just Works. The release workflow drops the replace before tagging.

When working on changes that span the root and an adapter, run tests in each module:

```
go test ./...                          # root module
( cd adapter/logrus && go test ./... ) # logrus module
( cd adapter/charm  && go test ./... ) # charm module
```

`go mod tidy` should also be run in each module that you touched.

## Tagging

Module-aware tags are required. Each module is tagged independently:

- root: `vX.Y.Z`
- logrus adapter: `adapter/logrus/vX.Y.Z`
- charm adapter: `adapter/charm/vX.Y.Z`

For the v0.1.0 cut, all three are tagged together. After that, modules may advance independently as long as the adapter modules remain compatible with the root interface contract.
