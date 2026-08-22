> [!CAUTION]
> This project is experimental and please do not use it.

# rizotto

## Tools

### rizotto make-project

Scaffolds a new project with the layout of [example](example), asking for everything
it needs step by step:

```sh
task make-project
# or
go install ./tools/rizotto && rizotto make-project
```

```
Project name (english letters and digits, e.g. MyShop): MyShop
Directory to create the project in [.]: ~/projects
Git repository (github.com/acme/shop): git@github.com:acme/shop.git
Does the project need SQL (PostgreSQL + sqlc + dbmate)? [y/N]: y
```

1. **Name** — english letters and digits only, so that it can be used as an
   identifier; it also becomes `SERVICE_NAME`, the RPC host and the database name.
2. **Directory** — the project is created in `<directory>/<name>`.
3. **Git repository** — any of `github.com/acme/shop`,
   `https://github.com/acme/shop.git` or `git@github.com:acme/shop.git`. The
   repository becomes the go module path, so every import inside the project
   follows it.
4. **SQL** — answering yes adds the repository layer with sqlc queries and dbmate
   migrations.

Every answer can be given upfront as a flag, and the matching question is then
skipped:

| flag            | meaning                                                                   |
| --------------- | ------------------------------------------------------------------------- |
| `-name`         | project name                                                              |
| `-path`         | directory the project folder is created in                                |
| `-repo`         | git repository, becomes the go module path                                |
| `-sql`          | `yes` or `no`                                                             |
| `-rizotto-path` | path to a local rizotto checkout; adds a `replace` directive to `go.mod`  |
| `-go-version`   | `go` directive of the generated `go.mod` (default: the running toolchain) |
| `-force`        | generate into an existing non-empty directory                             |
| `-skip-tidy`    | do not run `go mod tidy` in the created project                           |

The scaffold contains the settings, a sample `item` service with its client
bindings (the same code `service add` writes), the shared HTTP contexts in
`api/controller.go` and the OpenAPI document in `api/doc`. Controllers are added
separately with `rizotto controller add`.

Run `rizotto help` for the full list.

### rizotto service add

Adds a service to an existing project, asking what it should look like:

```sh
cd ~/projects/my-app && rizotto service add
# or from this repo
task service:add -- -project ~/projects/my-app
```

```
Service name (english letters and digits, e.g. Order): Order
Does the service work with the database (PostgreSQL)? [y/N]: y
CRUD methods (create, read, list, update, delete) [all]: create,read,list
```

The project is found by walking up from `-project` (default `.`) to the nearest
`go.mod`, whose module path is used for the imports. The command writes:

    services/order/order-service.go        the RPC implementation
    services/order/order-client/client.go  the contract, with the genbind directive
    services/order/order-client/bind-gen.go  the bindings, ready to be regenerated

and, when the service works with the database:

    services/order/repository/             repository, sqlc queries, generated db package
    services/order/repository/migrations/  the dbmate migration creating the table
    sqlc.yaml                              gets an entry for the new service

Without a database the service keeps its entities in memory, so it runs as is.
Afterwards the command prints the two lines to add to `server.go`; `task gen`
regenerates the bindings and `task migrate-up` applies the new migration.

| flag         | meaning                                                               |
| ------------ | --------------------------------------------------------------------- |
| `-name`      | service name, e.g. `Order`                                            |
| `-project`   | path inside the project the service is added to                       |
| `-db`        | `yes` when the service owns a database table                          |
| `-methods`   | `all` or a comma separated subset of `create,read,list,update,delete` |
| `-force`     | write into an existing service directory                              |
| `-skip-tidy` | do not run `go mod tidy` in the project                               |

### rizotto controller add

Adds an HTTP controller in front of one of the services:

```sh
cd ~/projects/my-app && rizotto controller add
# or from this repo
task controller:add -- -project ~/projects/my-app
```

```
Service to expose (item, order) [item]: order
Controller name [Order]: Order
Routes (create, read, list, update, delete) [create,read,list,update,delete]: read,list
Do the routes need an authenticated user (PrivateArea)? [Y/n]: y
Route prefix [/api/v1/orders]:
```

The services of the project are discovered by parsing their client interfaces, so
the routes are only offered for the RPCs a service really implements. The command
writes `api/order-api/order.go` (the controller, its JSON request and response
types and the error mapping) and `api/order-api/order_test.go` (a route test per
route, which also fills `api/doc` and writes `openapi.yml`), then prints the lines
to add to `server.go`.

The routes follow the chosen CRUD methods:

    POST   /api/v1/orders             -> CreateOrderRPC
    GET    /api/v1/orders/{order_id}  -> GetOrderRPC
    GET    /api/v1/orders             -> SelectOrdersRPC
    PUT    /api/v1/orders/{order_id}  -> UpdateOrderRPC
    DELETE /api/v1/orders/{order_id}  -> DeleteOrderRPC

| flag         | meaning                                                   |
| ------------ | --------------------------------------------------------- |
| `-service`   | service the controller exposes, e.g. `order`              |
| `-name`      | controller name (default: the name of the service)        |
| `-project`   | path inside the project the controller is added to        |
| `-methods`   | routes to generate (default: what the service implements) |
| `-auth`      | `yes` for `PrivateArea`, `no` for `PublicArea`            |
| `-prefix`    | route prefix (default `/api/v1/<plural name>`)            |
| `-force`     | write into an existing controller directory               |
| `-skip-tidy` | do not run `go mod tidy` in the project                   |

### rizotto skill

Prints the documentation of the framework to the console, as markdown:

```sh
rizotto skill              # what a rizotto project is made of: layout, bootstrap, settings, tasks
rizotto service skill      # services in detail: contract, genbind bindings, bb errors, repository, sqlc, migrations
rizotto controller skill   # controllers in detail: routing, request objects, contexts, error mapping, route tests
```

The texts live in [tools/rizotto/skills/](tools/rizotto/skills/) and are embedded in
the binary, so they work in a generated project too — handy as a briefing for a new
teammate or as context for an agent.

### genbind

Generates the service bindings from a client interface; see
[example/services/users/user-client/client.go](example/services/users/user-client/client.go)
for the `//go:generate genbind -client UserClient` directive.

## Example

An example server lives in [example](example). To run it from the repo root:

```sh
task example:start
```

It listens on the port set by the `PORT` env var (default `9090`). Configuration is read from `example/.env` (see [example/.env.example](example/.env.example) for the available variables).
