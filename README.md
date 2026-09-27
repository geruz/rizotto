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
   repository followed by `/server` becomes the go module path
   (`github.com/acme/shop/server`), so every import inside the project follows it.
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
| `-skip-git`     | do not run `git init` in the created project                              |

Unless `-skip-git` is given, the created project is initialised as an empty git
repository; the step is skipped when git is not installed or the target already
lives inside another repository.

The project is a repository of two parts:

    <name>/
      Taskfile.yml   forwards to the Taskfiles of server/ and web/
      server/        the go module, <repository>/server
      web/           the React app: Vite, TypeScript, Tailwind and shadcn/ui

`server/` contains the settings, a sample `item` service with its client
bindings (the same code `service add` writes), the shared HTTP contexts in
`api/controller.go` and the OpenAPI document in `api/doc`. Controllers are added
separately with `rizotto controller add`. The go module lives in `server/`, so its
path is the repository followed by `/server`.

`web/` is the result of `shadcn init` on a Vite app, with the `button` and `card`
components; `task web:dev` serves it on :5173 and proxies `/api` to the server.

`service add`, `controller add` and `solution add` work from anywhere in the
repository: walking up to the nearest `go.mod`, they also look into `server/`.

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

### rizotto solution add

Installs a whole feature rather than a layer. `service add` writes one service and
`controller add` writes one controller; a solution writes every layer the feature
is made of, and is opinionated about how they fit together:

```sh
rizotto solution list
rizotto solution add oauth -providers google,github
# or from this repo
task solution:add -- oauth -project ~/projects/my-app -providers google
```

The point is the features whose shape is not really a choice. Everybody's users
table looks the same, and getting the details of an OAuth flow wrong is expensive,
so these are the reviewed answer rather than one assembled per project.

**`user`** — the users of the project and their sessions:

    services/user/            the user service: users and sessions
    api/auth-area.go          AuthArea, which answers 401 before the handler runs
    api/user-api/             GET  /api/v1/me
                              POST /api/v1/logout

**`oauth`** — logging in through a provider, built on `user`:

    services/identity/        which external account belongs to which user
    services/identity/oauth/  the calls to the provider
    api/oauth-api/            GET /api/v1/auth/{provider}/start
                              GET /api/v1/auth/{provider}/callback

**`files`** — files on S3 compatible storage, built on `user`:

    services/file/            the metadata, the ownership and the size limit
    services/file/storage/    Signature Version 4, presigned links only
    api/file-api/             POST   /api/v1/files            announce, get an upload link
                              POST   /api/v1/files/{id}/confirm  check the bytes arrived
                              GET    /api/v1/files            the files of the caller
                              GET    /api/v1/files/{id}/content  where to read the bytes
                              DELETE /api/v1/files/{id}

The bytes never pass through the service: a caller PUTs the file to the presigned
link itself, which is also why no multipart parsing is needed anywhere. The object
key is generated, never built out of the name the caller sent.

Only the fingerprint of a session token is stored, never the token; a callback
whose state does not match the one we sent is refused; a file of another owner
reads as missing rather than forbidden. `rizotto solution skill <name>` covers the
tables, the flow and the configuration of each.

### Solutions built on solutions

A solution declares what it is built on, and those are installed first:

```sh
rizotto solution add oauth               # installs "user" too, when it is missing
```

A dependency the project already has is left alone. The flags only ever reach the
solution that was named, so a dependency is installed with its defaults and never
asks anything — install it by name first when the defaults are not what you want:

```sh
rizotto solution add user -session bearer
rizotto solution add oauth -providers github
```

### What a solution never does

It never edits `server.go` or `.env.example`, and never overwrites a file the
project already has: registering a service is one line that belongs to the author,
so the command finishes by printing exactly what to add and where. Files it would
overwrite are listed and refused unless `-force` says otherwise.

| flag         | meaning                                                        |
| ------------ | -------------------------------------------------------------- |
| `-project`   | path inside the project the solution is added to               |
| `-force`     | write the files even when the project already has some of them |
| `-skip-tidy` | do not run `go mod tidy` in the project                        |

`user` additionally takes `-session` (`cookie` for a browser application, `bearer`
for a native client), `oauth` takes `-providers` (`google`, `github`) and `files`
takes `-download` (`redirect` to the storage, or the link in the json answer).
Writing a new solution is described by `rizotto solution skill`.

### rizotto skill

Prints the documentation of the framework to the console, as markdown:

```sh
rizotto skill              # what a rizotto project is made of: layout, bootstrap, settings, tasks
rizotto service skill      # services in detail: contract, genbind bindings, bb errors, repository, sqlc, migrations
rizotto controller skill   # controllers in detail: routing, request objects, contexts, error mapping, route tests
rizotto solution skill     # what a solution is, how one is installed and how to write another
rizotto solution skill user  # users and sessions in detail
rizotto solution skill oauth # the OAuth login in detail: tables, flow, configuration
rizotto solution skill files # file storage in detail: the upload flow, the presigner
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
