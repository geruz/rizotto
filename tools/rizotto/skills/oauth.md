# OAuth

    rizotto solution add oauth -providers google,github

Installs the OAuth login: the identity service remembering which external account
belongs to which user, the calls to the providers, and the two routes performing
the login.

It **requires** the `user` solution, which owns the users and the sessions, and
installs it first with its defaults when the project does not have it yet. Install
`user` by name first if you want to answer its `-session` question yourself.

## What it writes

    services/identity/identity-client/client.go   the contract: 2 RPCs
    services/identity/identity-service.go         the implementation
    services/identity/repository/                 repository, queries, migration
    services/identity/oauth/provider.go           the calls to the providers
    api/oauth-api/oauth.go                        the two routes
    api/oauth-api/oauth_test.go                   the route tests

## The table

    user_identities   id, user_id, provider, external_id, ... UNIQUE(provider, external_id)

The pair of `provider` and `external_id` is what identifies a caller. `user_id`
carries **no foreign key**: the users belong to the user service, which owns its
own tables and its own migration history, and a constraint across that boundary
would tie the two together and force one migration order on them.

## The routes

    GET /api/v1/auth/{provider}/start      302 to the provider, sets the state cookie
    GET /api/v1/auth/{provider}/callback   302 back to the app, hands out the session

`start` generates a state token, remembers it in a short lived cookie and sends the
caller to the provider. `callback` refuses anything whose state does not match that
cookie, which is what keeps a third party from logging our callers in as somebody
else. It answers the same error whatever went wrong, so it cannot be used to find
out which codes or states exist.

`/logout` and `/me` are not here: they are about sessions, not about logging in,
and belong to the `user` solution.

## How the two solutions meet

The controller orchestrates, and the services stay independent — neither calls the
other:

    identity.GetIdentityRPC      is this external account known?
    user.CreateUserRPC           no: make the user
    identity.CreateIdentityRPC   and link the account to it
    user.CreateSessionRPC        issue the session
    api.HandOverSession          hand it to the caller, however sessions travel here

The last line is the whole coupling. This solution never learns whether the session
went out as a cookie or on a fragment.

## Configuration

    OAUTH_<PROVIDER>_CLIENT_ID       the OAuth client
    OAUTH_<PROVIDER>_CLIENT_SECRET
    OAUTH_REDIRECT_BASE_URL          where the provider sends the caller back

The redirect URI registered with the provider has to be
`<OAUTH_REDIRECT_BASE_URL>/api/v1/auth/<provider>/callback`.

The three endpoints of a provider default to the real ones and can be pointed
elsewhere with `OAUTH_<PROVIDER>_AUTHORIZE_URL`, `_TOKEN_URL` and
`_USER_INFO_URL`. The generated tests use that to run the whole flow against a fake
provider inside the process; a staging environment can use it too.

## Adding a provider

`knownProviders` in `tools/rizotto/solution_oauth.go` is the table: an entry is the
three endpoints, the scope, and the names of the id, email and name fields of the
user info answer. The exchange is plain `net/http`, so a provider that follows the
authorization code flow needs no code, only a row.

## What it deliberately leaves out

No PKCE — the flow runs server side with a client secret, where PKCE adds nothing.
No refresh tokens: the provider's access token is used once, to ask who the caller
is, and then dropped; the session of this project is what lasts. No account
linking: two providers answering for the same person produce two users, and merging
them is a decision only the project can make.
