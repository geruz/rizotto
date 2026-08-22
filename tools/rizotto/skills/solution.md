# Solutions

A solution is a whole feature rather than a layer. `service add` writes one
service, `controller add` writes one controller; a solution writes every layer a
feature is made of at once, and it is opinionated about how they fit together.

    rizotto solution list                  # what can be installed
    rizotto solution add oauth [flags]     # install one
    rizotto solution skill oauth           # what that one does

## When to use one

Use `service add` when the entities are yours to design: an order, an invoice, a
template. Use a solution when the shape is not really a choice — everybody's
users table looks the same, and getting the details of an OAuth flow wrong is
expensive. A solution is the answer that has already been reviewed.

## What installing one does

The command finds the project by walking up to the nearest `go.mod`, reads back
what the project supports, and writes:

- the services it needs, with their repositories, sqlc queries and dbmate
  migrations, registering each in `sqlc.yaml`;
- the controllers, with their route tests;
- whatever else the feature is made of, such as the area function of an
  authenticated route.

Migrations are picked up by the `migrate` task on their own: it globs
`services/*/repository/migrations` and gives each service its own migrations
table, so nothing has to be registered.

## Solutions built on solutions

A solution may declare the ones it is built on, and they are installed first:

    rizotto solution add oauth      # installs "user" too, when the project lacks it

A dependency the project already has is left alone — it is recognised by a file of
its own, so nothing extra is written down to track it.

The flags on the command line only ever reach the solution that was named. A
dependency is installed with its **defaults**, which is why every question a
solution asks must have one. When those defaults are not what you want, install the
dependency by name first:

    rizotto solution add user -session bearer
    rizotto solution add oauth -providers github

Circular requirements are an error rather than a hang, and a solution requiring a
name that is not in the registry fails before anything is written.

## What it does not do

It never edits `server.go`, `.env.example` or a file the project already has.
Registering a service and joining a route table are one line each, they belong to
the author, and a generator guessing at them is worse than a generator printing
them. So the command finishes by printing exactly what to add and where. The
files it would overwrite are refused with a list, unless `-force` says otherwise.

## Flags

Every solution takes these:

| flag         | meaning                                                   |
| ------------ | --------------------------------------------------------- |
| `-project`   | path inside the project to install into (default `.`)     |
| `-force`     | write the files even when the project already has some    |
| `-skip-tidy` | do not run `go mod tidy` afterwards                       |

Plus its own, which `rizotto solution add <name> -h` lists. Passing an answer as a
flag skips the matching question, so the command never goes interactive when every
flag is given — which is what an agent should do.

## Adding a solution to rizotto

A solution is a Go file in `tools/rizotto`, next to `solution_auth.go`, and a
directory of templates under `templates/solutions/<name>`. It implements:

```go
type solution interface {
    Name() string                       // "oauth"
    Summary() string                    // the line "solution list" prints
    Usage() string                      // the help of "solution add oauth"
    Skill() string                      // the skills/<name>.md document
    Requires() []string                 // the solutions it is built on
    Marker() string                     // a file that exists once it is installed
    BindFlags(fs *flag.FlagSet) any     // its flags, returned for Ask
    Ask(flags any, pr *prompter, prj projectInfo) (solutionPlan, error)
}
```

`Ask` does not write anything: it answers a `solutionPlan` describing the
installation, and one shared runner performs it. That is what makes every solution
behave the same way about conflicts, reporting and `go mod tidy`.

```go
type solutionPlan struct {
    Data         any                 // what the templates are rendered with
    Files        []fileSpec          // the templates of the solution
    Services     []serviceSpec       // generated the usual way, sqlc included
    Controllers  []controllerIntent  // resolved once the services exist
    Repositories []sqlcEntry         // repositories not belonging to a Service
    Env          []envVar            // printed for .env.example
    Steps        []string            // printed as the manual wiring
}
```

Two things are worth knowing when writing one:

- A `controllerIntent` is resolved **after** the services are written, because a
  controller spec needs the `serviceInfo` that only comes from parsing the
  `client.go` of a service that already exists on disk.
- CRUD shaped services should go through `Services`, which gets the migrations
  and the sqlc entry for free. A service whose RPCs are not
  `Create`/`Get`/`Select`/`Update`/`Delete` brings its own templates in `Files`
  and its own `Repositories` entry, as `user` and `oauth` do.

Two rules keep dependent solutions from growing into each other:

- **Each solution owns its own tables and its own migrations.** Never write into
  the migrations directory of another one. A column pointing at a table of another
  solution carries no foreign key, because that would tie their migration
  histories together — `user_identities.user_id` is the example.
- **Talk through the contract, not the files.** The services of a solution stay
  independent; a controller orchestrates them. Where a decision belongs to the
  other solution, it exports a function for it: `api.HandOverSession` is how
  `oauth` hands out a session without knowing how sessions travel.

Finally, add the solution to the `solutions` registry, write its
`skills/<name>.md`, and mention it in the `SKILL.md` template so that the agents
working in generated projects can find it.
