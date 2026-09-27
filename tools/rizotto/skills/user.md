# Users

    rizotto solution add user -session cookie -rbac

Installs the users of the project and their sessions: the user service, the area
function turning a session into the current user of a private route, and the two
routes every session needs. The project must have `sqlc.yaml`, which make-project writes.

This solution knows nothing about how somebody logs in. `solution add oauth` adds
that on top; a project with its own way of authenticating people — an internal
directory, a magic link, a password — uses this one alone and calls
`CreateUserRPC` and `CreateSessionRPC` itself. Since `oauth` never registers
anybody, creating the users with `CreateUserRPC` is the project's job either way.

## What it writes

    services/user/user-client/client.go     the contract: 6 RPCs
    services/user/user-client/bind-gen.go   the bindings
    services/user/user-service.go           the implementation
    services/user/repository/               repository, queries, migration
    api/auth-area.go                        AuthArea, AuthHTTPContext, session transport
    api/user-api/user.go                    GET /api/v1/me, POST /api/v1/logout
    api/user-api/user_test.go               the route tests

## The tables

    users      id, email, name, created_at, updated_at
    sessions   id, token_hash UNIQUE, user_id, expires_at, ...

The email is **not** unique: how a person is identified is the business of whatever
authenticates them. The same person may reach the project through two providers,
and a provider may hand out an address another one already used.

`GetUserByEmailRPC` finds a user by email, ignoring case. When several users share
the email it does not pick one: it answers a validation error with the code
`user.SharedEmailCode`, and the caller has to tell them apart some other way.

Only the fingerprint of a session token is stored, never the token itself, so a
leaked database hands over no working sessions. `crypto/token.Secure` generates the
token and `crypto/token.Hash` reduces it. `CreateSessionRPC` is the only answer
that ever carries the token.

## Authenticating your own routes

`AuthArea` is the real area function. It reads the session and answers 401 by
itself when there is none — the handler is never reached:

```go
gateway.JSONMethod("GET /api/v1/orders", api.AuthArea, ctrl.listOrders)

func (ctrl OrderController) listOrders(
    ctx api.AuthHTTPContext, req OrderListRequest,
) (OrderList, gateway.HTTPError) {
    // ctx.User.ID is there: the request would not have reached this line otherwise
}
```

`api/controller.go` is left alone, placeholder and all, so installing the solution
never rewrites a file the project has edited. Its `PrivateArea` keeps returning a
hardcoded user; move a route over by swapping `api.PrivateArea` for `api.AuthArea`
and `api.UserHTTPContext` for `api.AuthHTTPContext`.

## Handing out a session

`api/auth-area.go` owns how a session travels, so nothing else has to know:

```go
location := api.HandOverSession(ctx, session.Token, session.ExpiresAt)  // where to send the caller
api.DropSession(ctx)                                                   // take it away again
token, found := api.SessionToken(ctx)                                  // read it back
```

That is the seam the oauth solution uses, and the one your own login should use.

## `-session cookie` or `-session bearer`

`cookie` puts the session in an `HttpOnly`, `Secure`, `SameSite=Lax` cookie, and
`HandOverSession` redirects to `APP_LOGIN_REDIRECT_URL`. Right for a browser
application: JavaScript never touches the token.

`bearer` expects `Authorization: Bearer <token>`, and `HandOverSession` appends the
token to the fragment of the redirect, which browsers keep out of both requests and
server logs. Right for a native client or a separate front end.

The answer is baked into the generated `api/auth-area.go`, so it is chosen once.
Installing `oauth` first would install this solution with the `cookie` default —
install `user` by name first when you want `bearer`.

## Configuration

    APP_LOGIN_REDIRECT_URL    where the caller lands once the session exists
    SESSION_TTL_HOURS         how long a session lasts (default 720)

## Groups, roles and permissions: `-rbac`

Off by default. With it, a user is granted what the roles of their groups are
granted:

    user ── member of ──> group ── granted ──> role ── holds ──> permission

Roles reach users **only through groups**; to give one person a role, make a group
for them. Everything is global: a permission holds for the whole project, not for
one organization or one record.

It writes, on top of the rest:

    permissions/permissions.go                        the permissions of the project
    services/user/access-client/                      the access contract: 15 RPCs
    services/user/access-service.go                   its implementation, and the admins bootstrap
    services/user/repository/access-repository.go
    services/user/repository/migrations/000002_create_access.sql
    api/accesstest/accesstest.go                      a stand-in for route tests
    api/access-api/                                   the admin routes and their tests

### Permissions live in the code, roles in the database

A permission is a constant of `permissions/permissions.go`, because the code
checking it is what gives it a meaning. Roles are rows naming sets of those, so the
admins change them without a deployment. A role granted a permission the code does
not declare is refused (`unknown_permission`): it could never be checked anywhere.

```go
const (
    UsersManage Permission = "users.manage"
    OrdersWrite Permission = "orders.write" // add yours here...
)

func All() []Permission {
    return []Permission{UsersManage, OrdersWrite} // ...and here
}
```

### Guarding a route

`api.Require` builds the area function: the caller has to be authenticated, as
with `AuthArea`, and granted every permission listed, or the request is answered
403 before the handler runs.

```go
gateway.JSONMethod("POST /api/v1/orders", api.Require(permissions.OrdersWrite), ctrl.create)

func (ctrl OrderController) create(ctx api.AuthHTTPContext, req CreateOrderRequest) (Order, gateway.HTTPError) {
    if ctx.User.Can(permissions.OrdersApprove) { ... } // finer checks inside the handler
}
```

`ctx.User.Permissions` is only read by `Require`: behind a plain `AuthArea` it is
empty and `Can` answers false. `api.Require()` with no permission requires nothing
and only reads them — `GET /api/v1/me` uses it to answer `permissions`, so the
application can hide what the caller cannot do.

The permissions are read on every request guarded by `Require`, one query joining
three tables. There is no cache, so a change of a role applies at once.

### The admin routes

Every route of `api/access-api` requires `permissions.UsersManage`.

    GET    /api/v1/admin/permissions                              what a role can be granted
    GET    /api/v1/admin/users?offset&limit
    POST   /api/v1/admin/users                                    register a user
    GET    /api/v1/admin/roles
    POST   /api/v1/admin/roles                                    {name, permissions}
    PUT    /api/v1/admin/roles/{role_id}                          renames, replaces the permissions
    DELETE /api/v1/admin/roles/{role_id}
    GET    /api/v1/admin/groups                                   with their roleIds and memberIds
    GET    /api/v1/admin/groups/{group_id}
    POST   /api/v1/admin/groups                                   {name}
    PUT    /api/v1/admin/groups/{group_id}                        {name}
    DELETE /api/v1/admin/groups/{group_id}
    PUT    /api/v1/admin/groups/{group_id}/members/{user_id}
    DELETE /api/v1/admin/groups/{group_id}/members/{user_id}
    PUT    /api/v1/admin/groups/{group_id}/roles/{role_id}
    DELETE /api/v1/admin/groups/{group_id}/roles/{role_id}

Role and group names are unique (`409 name_taken`). Adding a member or granting a
role twice is not an error, and neither is removing one that is not there.

### The first admin: `ADMIN_EMAILS`

`AccessService.MustBootstrapAdmins`, called from `server.go` before anything is
served, puts every user of `ADMIN_EMAILS` into the `admins` group, whose `admin`
role is granted every permission of `permissions.All()`. A user missing yet is
created, named after their email. It runs at every start, so:

- a permission added to the code reaches the admins with the next deployment;
- admins who removed themselves are back after a restart; take an email out of
  `ADMIN_EMAILS` to really revoke it;
- an email several users share stops the start: nobody can tell which one is meant.

With `ADMIN_EMAILS` empty it does nothing at all.

### Why a contract of its own

The RPCs are in `access.AccessClient`, served by the same user service, rather than
in `user.UserClient`. The `oauth` and `files` solutions stub `UserClient` in their
tests, and those stubs stay the same whether the project has `-rbac` or not.

A route test behind `Require` registers `accesstest.Register(grants)` in `TestMain`,
next to its stub of the user service, to say which user is granted what.

## What it deliberately leaves out

No passwords: `crypto/password` is there if you add them. No refresh tokens — a
session has one lifetime and logging in again is the renewal. Without `-rbac` no
roles or scopes; with it no per-resource permissions and no roles given to a user
directly. Expired sessions are rejected on read but never deleted, so a periodic
cleanup is yours to add.
