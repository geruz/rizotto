# rizotto

rizotto is a small Go framework for HTTP services. A project built with it is made
of three kinds of building blocks:

- **services** — the business logic, reachable through generated RPC clients;
- **controllers** — the HTTP layer translating requests into RPC calls;
- **the server** — `server.go`, which wires the settings, the services and the routes.

Everything else (settings, logging, metrics, tracing, database access, tests and
OpenAPI documentation) comes from the framework packages.

## Commands

    rizotto make-project        create a new project (name, directory, git repository, SQL)
    rizotto service add         add a service (database, CRUD methods)
    rizotto controller add      add an HTTP controller for one of the services
    rizotto skill               this text
    rizotto service skill       how a service works, in detail
    rizotto controller skill    how a controller works, in detail

## Project layout

    server.go                       bootstrap: settings, services, routes
    .claude/skills/rizotto/         the Claude Code skill pointing back at these commands
    .env / .env.example             configuration, .env imports .env.example
    Taskfile.yml                    task deps | start | test | lint | gen | migrate-up
    sqlc.yaml                       one entry per service that owns a table
    api/controller.go               HTTP contexts shared by the controllers
    api/doc/                        the OpenAPI document the route tests fill in
    api/<name>-api/                 one controller: routes, DTOs, route tests
    services/<name>/                one service: the RPC implementation
    services/<name>/<name>-client/  its contract; bind-gen.go is generated from client.go
    services/<name>/repository/     sqlc queries, generated db package, dbmate migrations

## The bootstrap in server.go

```go
func main() {
    ctx := context.Background()
    rizotto.MustInitEnv(ctx)     // loads .env (and the files named in IMPORT_ENV_FILES)
    rizotto.MustInitLogger(ctx)  // LOG_LEVEL
    rizotto.MustInitMetrics(ctx, ":"+env.GetStringValue("METRICS_PORT", "9091")) // /metrics
    rizotto.MustInitTracer(ctx, map[string]string{ // OTLP, configured by OTEL_* variables
        "service.name":    env.MustGetStringValue("SERVICE_NAME"),
        "service.version": env.MustGetStringValue("SERVICE_VERSION"),
        "environment":     env.MustGetStringValue("ENVIRONMENT"),
    })

    gt := gateway.NewHTTPGateway()

    pgPool := pg.MustOpenConnection(ctx, env.MustGetStringValue("DATABASE_URL")) // only with SQL

    order.RegisterServer(ordersrv.NewOrderService(pgPool)) // one line per service

    err := gt.Routing(
        gateway.JoinRouteTables(
            orderapi.NewOrderController().RouteTable(), // one line per controller
        ),
    ).ListenAndServe(ctx, ":"+env.GetStringValue("PORT", "9090"))
    if err != nil {
        panic(err)
    }
}
```

Services are registered before the routes are bound: a controller binds its client
in its constructor, and the binding resolves the RPC registered by the service.

## Settings

`settings/env` reads the process environment. `.env` is loaded first and pulls in
the files listed in `IMPORT_ENV_FILES` (`.env.example` by default); a value already
present in the environment always wins, so `.env.example` holds the defaults and
`.env` the local overrides.

```go
env.MustGetStringValue("DATABASE_URL")        // panics when missing
env.GetStringValue("PORT", "9090")            // with a default
env.GetIntValue("WORKERS", 4)
env.MustGetBoolValue("FEATURE_X")
env.MustGetEnumValue[Mode]("MODE", []Mode{ModeA, ModeB})
```

## Logging, metrics, tracing

```go
logger.Info(ctx, "message", logger.KV("order_id", id))
logger.Error(ctx, "failed to do the thing", err, logger.KVi("order_id", 42))

var ordersTotal = metrics.Counter("orders_total", "Created orders.")
var duration = metrics.DurationHistogram("orders_seconds", "…", metrics.DurationBuckets_5ms_10s)
```

The logger prints the trace id of the current span, so log lines and traces line up.
Metrics are served on `METRICS_PORT` at `/metrics`; traces go to the OTLP endpoint
from the standard `OTEL_EXPORTER_OTLP_*` variables.

## Everyday tasks

    task deps          go mod tidy
    task start         go run .
    task test          go test ./... with coverage; refreshes every openapi.yml
    task lint          golangci-lint run ./...
    task gen           regenerate bind-gen.go (genbind) and the sqlc queries (docker)
    task migrate-up    apply the dbmate migrations of every service
    task migrate-down  roll the last migration of every service back

`task gen` needs the `genbind` binary: `go install github.com/geruz/rizotto/tools/genbind@latest`.

Run `rizotto service skill`, `rizotto controller skill` and
`rizotto repository skill` for the details of each layer.
