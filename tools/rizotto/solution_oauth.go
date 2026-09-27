package main

import (
	"errors"
	"flag"
	"fmt"
	"slices"
	"strings"
)

const oauthUsage = `Usage:
	rizotto solution add oauth [flags]

Installs the OAuth login: the identity service remembering which external account
belongs to which user, the calls to the providers, and the two routes performing
the login. It only logs users in, never registers them: the first login of an
account links it to the existing user with the email the provider vouches for.

It is built on the "user" solution, which owns the users and the sessions, and
installs it first when the project does not have it yet.

Flags:
	-providers   comma separated OAuth providers: google, github (default "google")

The project must have sqlc.yaml (make-project writes it), since the solution owns a table.
`

// authProvider is one OAuth provider and the endpoints it is reached at.
type authProvider struct {
	// Key is the value of the {provider} path variable, e.g. google.
	Key string
	// Name is the exported name used in the generated identifiers, e.g. Google.
	Name string
	// EnvPrefix names the variables holding the credentials.
	EnvPrefix string
	// AuthorizeURL is where the caller is sent to log in.
	AuthorizeURL string
	// TokenURL exchanges the authorization code for an access token.
	TokenURL string
	// UserInfoURL answers who the access token belongs to.
	UserInfoURL string
	// Scope is what the login asks for.
	Scope string
	// IDField, EmailField and NameField are the fields of the user info answer.
	IDField    string
	EmailField string
	NameField  string
	// VerifiedField is the field telling whether the provider checked the email.
	// Empty means the provider only ever answers checked addresses.
	VerifiedField string
}

const (
	providerGoogle = "google"
	providerGitHub = "github"

	defaultProviders = providerGoogle
)

//nolint:gochecknoglobals,gosec // the table is static; G101 reads TokenURL as a credential
var knownProviders = []authProvider{
	{
		Key:           providerGoogle,
		Name:          "Google",
		EnvPrefix:     "OAUTH_GOOGLE",
		AuthorizeURL:  "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:      "https://oauth2.googleapis.com/token",
		UserInfoURL:   "https://openidconnect.googleapis.com/v1/userinfo",
		Scope:         "openid email profile",
		IDField:       "sub",
		EmailField:    "email",
		NameField:     "name",
		VerifiedField: "email_verified",
	},
	{
		Key:          providerGitHub,
		Name:         "GitHub",
		EnvPrefix:    "OAUTH_GITHUB",
		AuthorizeURL: "https://github.com/login/oauth/authorize",
		TokenURL:     "https://github.com/login/oauth/access_token",
		UserInfoURL:  "https://api.github.com/user",
		Scope:        "read:user user:email",
		IDField:      "id",
		EmailField:   "email",
		NameField:    "name",
		// The public email of a GitHub account can only be a verified one.
		VerifiedField: "",
	},
}

var (
	errNoProviders     = errors.New("choose at least one OAuth provider: google, github")
	errUnknownProvider = errors.New("unknown OAuth provider")
)

// oauthSpec is the data the templates of the oauth solution are rendered with.
type oauthSpec struct {
	Module        string
	RizottoModule string
	// Providers are the chosen OAuth providers.
	Providers []authProvider
	// StateCookieName is the cookie carrying the OAuth state between the two legs.
	StateCookieName string
	// Methods drive the client interface and the generated bindings.
	Methods []serviceMethod
}

func (o oauthSpec) Dir() string {
	return "services/identity"
}

func (o oauthSpec) ClientImport() string {
	return o.Module + "/" + o.Dir() + "/identity-client"
}

// UserClientImport is the contract of the user solution, which owns the users and
// the sessions the login ends up creating.
func (o oauthSpec) UserClientImport() string {
	return o.Module + "/services/user/user-client"
}

func (o oauthSpec) APIImport() string {
	return o.Module + "/api"
}

func (o oauthSpec) OAuthImport() string {
	return o.Module + "/" + o.Dir() + "/oauth"
}

func identityMethods() []serviceMethod {
	address := func(action string) string {
		return "http://identity-service/identity/" + action
	}

	return []serviceMethod{
		{Name: "GetIdentityRPC", Request: "GetIdentityRequest", Response: "Identity", Address: address("get")},
		{
			Name:     "CreateIdentityRPC",
			Request:  "CreateIdentityRequest",
			Response: "Identity",
			Address:  address("create"),
		},
	}
}

type oauthSolution struct{}

func (oauthSolution) Name() string {
	return oauthSolutionName
}

func (oauthSolution) Summary() string {
	return "log in through an OAuth provider, on top of the user solution"
}

func (oauthSolution) Usage() string {
	return oauthUsage
}

func (oauthSolution) Skill() string {
	return oauthSolutionName
}

func (oauthSolution) Requires() []string {
	return []string{userSolutionName}
}

func (oauthSolution) Marker() string {
	return "services/identity/identity-service.go"
}

type oauthFlags struct {
	providers string
}

func (oauthSolution) BindFlags(fs *flag.FlagSet) any {
	f := &oauthFlags{} //nolint:exhaustruct_v5 // filled by the flag package

	fs.StringVar(&f.providers, "providers", "",
		`comma separated OAuth providers: google, github (default "google")`)

	return f
}

func (o oauthSolution) Ask(flags any, pr *prompter, prj projectInfo) (solutionPlan, error) {
	f, ok := flags.(*oauthFlags)
	if !ok {
		return solutionPlan{}, fmt.Errorf("%w: %T", errWrongFlags, flags)
	}

	if !prj.SQL {
		return solutionPlan{}, errSolutionNeedsSQL
	}

	providers, err := answer(pr, f.providers,
		"OAuth providers ("+providerKeys()+")", defaultProviders, normalizeProviders)
	if err != nil {
		return solutionPlan{}, err
	}

	spec := oauthSpec{
		Module:          prj.Module,
		RizottoModule:   rizottoModule,
		Providers:       chosenProviders(providers),
		StateCookieName: "oauth_state",
		Methods:         identityMethods(),
	}

	return solutionPlan{
		Data:         spec,
		Files:        spec.files(),
		Services:     nil,
		Controllers:  nil,
		Repositories: []sqlcEntry{newSqlcEntry(spec.Dir(), "identity-queries.sql")},
		Env:          spec.env(),
		Steps:        spec.steps(),
	}, nil
}

// normalizeProviders turns "google, gh" into the canonical "google,github".
func normalizeProviders(raw string) (string, error) {
	fields := strings.FieldsFunc(strings.ToLower(strings.TrimSpace(raw)), func(r rune) bool {
		return r == ',' || r == ' ' || r == '+' || r == ';'
	})

	chosen := map[string]bool{}

	for _, field := range fields {
		key, ok := providerKey(field)
		if !ok {
			return "", fmt.Errorf("%w %q, known providers: %s", errUnknownProvider, field, providerKeys())
		}

		chosen[key] = true
	}

	if len(chosen) == 0 {
		return "", errNoProviders
	}

	selected := make([]string, 0, len(knownProviders))

	for _, provider := range knownProviders {
		if chosen[provider.Key] {
			selected = append(selected, provider.Key)
		}
	}

	return strings.Join(selected, ","), nil
}

func providerKey(field string) (string, bool) {
	aliases := map[string]string{
		providerGoogle: providerGoogle,
		"g":            providerGoogle,
		providerGitHub: providerGitHub,
		"gh":           providerGitHub,
	}

	key, ok := aliases[field]

	return key, ok
}

func providerKeys() string {
	keys := make([]string, 0, len(knownProviders))
	for _, provider := range knownProviders {
		keys = append(keys, provider.Key)
	}

	return strings.Join(keys, ", ")
}

func chosenProviders(answer string) []authProvider {
	keys := strings.Split(answer, ",")
	providers := make([]authProvider, 0, len(keys))

	for _, provider := range knownProviders {
		if slices.Contains(keys, provider.Key) {
			providers = append(providers, provider)
		}
	}

	return providers
}

func (o oauthSpec) files() []fileSpec {
	dir := o.Dir()

	return []fileSpec{
		{tmpl: "solutions/oauth/client.go.tmpl", out: dir + "/identity-client/client.go"},
		{tmpl: "solutions/oauth/bind-gen.go.tmpl", out: dir + "/identity-client/bind-gen.go"},
		{tmpl: "solutions/oauth/identity-service.go.tmpl", out: dir + "/identity-service.go"},
		{tmpl: "solutions/oauth/repository.go.tmpl", out: dir + "/repository/identity-repository.go"},
		{tmpl: "solutions/oauth/schema.sql.tmpl", out: dir + "/repository/sql/schema.sql"},
		{tmpl: "solutions/oauth/queries.sql.tmpl", out: dir + "/repository/sql/identity-queries.sql"},
		{
			tmpl: "solutions/oauth/migration.sql.tmpl",
			out:  dir + "/repository/migrations/000001_create_user_identities.sql",
		},
		{tmpl: "solutions/oauth/db/db.go.tmpl", out: dir + "/repository/db/db.go"},
		{tmpl: "solutions/oauth/db/models.go.tmpl", out: dir + "/repository/db/models.go"},
		{tmpl: "solutions/oauth/db/queries.sql.go.tmpl", out: dir + "/repository/db/identity-queries.sql.go"},
		{tmpl: "solutions/oauth/provider.go.tmpl", out: dir + "/oauth/provider.go"},
		{tmpl: "solutions/oauth/oauth-api.go.tmpl", out: "api/oauth-api/oauth.go"},
		{tmpl: "solutions/oauth/oauth-api_test.go.tmpl", out: "api/oauth-api/oauth_test.go"},
	}
}

func (o oauthSpec) env() []envVar {
	vars := []envVar{}

	for _, provider := range o.Providers {
		vars = append(vars,
			envVar{
				Key:     provider.EnvPrefix + "_CLIENT_ID",
				Value:   "",
				Comment: "the OAuth client of " + provider.Name,
			},
			envVar{Key: provider.EnvPrefix + "_CLIENT_SECRET", Value: "", Comment: ""},
		)
	}

	return append(vars, envVar{
		Key:     "OAUTH_REDIRECT_BASE_URL",
		Value:   "http://localhost:9090",
		Comment: "where the provider sends the caller back, without the path",
	})
}

func (o oauthSpec) steps() []string {
	register := fmt.Sprintf(`register the service in server.go, before the controllers:

         import (
             "%s"
             identitysrv "%s/%s"
         )

         identity.RegisterServer(identitysrv.NewIdentityService(pgPool))`,
		o.ClientImport(), o.Module, o.Dir())

	routes := `join the route table in server.go:

         import "` + o.Module + `/api/oauth-api"

         gt.Routing(
             gateway.JoinRouteTables(
                 oauthapi.NewOAuthController().RouteTable(),
             ),
         )`

	redirect := `register the redirect URI with every provider:

         <OAUTH_REDIRECT_BASE_URL>/api/v1/auth/<provider>/callback`

	return []string{
		stepEnvExample,
		register,
		routes,
		redirect,
		stepGen,
		"task migrate-up # applies the table of the solution",
		"task test       # runs the route tests and writes api/oauth-api/openapi.yml",
	}
}
