# go-logger

A small logging interface for Go and a set of forward adapters that route calls
into popular underlying loggers.

## Interface

The root module defines `iface.Logger`, `iface.Controller`, `iface.Fields`, and
related interfaces. Adapters expose values that satisfy these.

## Adapters

| Adapter | Import path | Module | Notes |
|---|---|---|---|
| discard | `github.com/anchore/go-logger/adapter/discard` | root | drops everything; no external deps |
| redact | `github.com/anchore/go-logger/adapter/redact` | root | redacts configured tokens before delegating |
| slog | `github.com/anchore/go-logger/adapter/slog` | root | wraps stdlib `log/slog` |
| logrus | `github.com/anchore/go-logger/adapter/logrus` | own module | wraps `sirupsen/logrus` |
| charm | `github.com/anchore/go-logger/adapter/charm` | own module | wraps `charmbracelet/log` |

The `logrus` and `charm` adapters live in their own Go modules so that
consumers using one (or just the interface) do not pull the other's transitive
dependencies into their `go.sum`. Importing them works exactly the same as
importing any other Go module — `go mod tidy` will record a `require` line for
the adapter module on first use.

### slog: importing alongside `log/slog`

The adapter package is named `slog`, the same as the stdlib package. Callers
that import both will need to alias one of them:

```go
import (
    "log/slog"
    slogadapter "github.com/anchore/go-logger/adapter/slog"
)
```

### Trace level

`iface.TraceLevel` has no native equivalent in `log/slog` or `charmbracelet/log`,
so each adapter exposes a custom `LevelTrace` constant (one step below their
respective debug levels). When wrapping a pre-built logger via `Use(...)`, the
caller is responsible for configuring their handler/logger to permit
`LevelTrace` if trace logging is desired.
