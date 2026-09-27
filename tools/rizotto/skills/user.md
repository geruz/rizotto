# Users

    rizotto solution add user -session cookie

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

## What it deliberately leaves out

No passwords: `crypto/password` is there if you add them. No refresh tokens — a
session has one lifetime and logging in again is the renewal. No roles or scopes;
authorization is the project's business, and `gateway.NewForbiddenError` is what
answers it. Expired sessions are rejected on read but never deleted, so a periodic
cleanup is yours to add.
