package main

import (
	"errors"
	"flag"
	"fmt"
	"strings"
)

const userUsage = `Usage:
	rizotto solution add user [flags]

Installs the users of the project and their sessions: the user service, the area
function turning a session into the current user of a private route, and the two
routes every session needs.

It knows nothing about how somebody logs in. "solution add oauth" adds that on
top; a project with its own way of authenticating people uses this one alone.

Flags:
	-session     where the session token travels: "cookie" or "bearer" (default "cookie")

The project must have sqlc.yaml (make-project writes it), since the solution owns tables.
`

const (
	sessionCookie = "cookie"
	sessionBearer = "bearer"
)

var errSessionAnswer = errors.New(`-session expects "cookie" or "bearer"`)

// userSpec is the data the templates of the user solution are rendered with.
type userSpec struct {
	Module        string
	RizottoModule string
	// Cookie tells whether the session travels in a cookie rather than a header.
	Cookie bool
	// SessionCookieName is the cookie carrying the session.
	SessionCookieName string
	// Methods drive the client interface and the generated bindings, so that
	// genbind reproduces bind-gen.go from the contract.
	Methods []serviceMethod
}

func (u userSpec) Dir() string {
	return "services/user"
}

func (u userSpec) ClientImport() string {
	return u.Module + "/" + u.Dir() + "/user-client"
}

func (u userSpec) APIImport() string {
	return u.Module + "/api"
}

// userMethods are the RPCs of the user service. They are not CRUD shaped, which
// is why the solution brings its own service templates.
func userMethods() []serviceMethod {
	address := func(entity, action string) string {
		return "http://user-service/" + entity + "/" + action
	}

	const userResponse = "User"

	return []serviceMethod{
		{Name: "GetUserRPC", Request: "GetUserRequest", Response: userResponse, Address: address("user", "get")},
		{
			Name:     "GetUserByEmailRPC",
			Request:  "GetUserByEmailRequest",
			Response: userResponse,
			Address:  address("user", "get-by-email"),
		},
		{Name: "CreateUserRPC", Request: "CreateUserRequest", Response: userResponse, Address: address("user", "create")},
		{
			Name:     "CreateSessionRPC",
			Request:  "CreateSessionRequest",
			Response: "Session",
			Address:  address("session", "create"),
		},
		{Name: "GetSessionRPC", Request: "GetSessionRequest", Response: "Session", Address: address("session", "get")},
		{
			Name:     "DeleteSessionRPC",
			Request:  "DeleteSessionRequest",
			Response: "DeletedSession",
			Address:  address("session", "delete"),
		},
	}
}

type userSolution struct{}

func (userSolution) Name() string {
	return userSolutionName
}

func (userSolution) Summary() string {
	return "the users of the project and their sessions, with the area function authenticating private routes"
}

func (userSolution) Usage() string {
	return userUsage
}

func (userSolution) Skill() string {
	return userSolutionName
}

func (userSolution) Requires() []string {
	return nil
}

func (userSolution) Marker() string {
	return "services/user/user-service.go"
}

type userFlags struct {
	session string
}

func (userSolution) BindFlags(fs *flag.FlagSet) any {
	f := &userFlags{} //nolint:exhaustruct_v5 // filled by the flag package

	fs.StringVar(&f.session, "session", "",
		`where the session token travels: "cookie" or "bearer" (default "cookie")`)

	return f
}

func (u userSolution) Ask(flags any, pr *prompter, prj projectInfo) (solutionPlan, error) {
	f, ok := flags.(*userFlags)
	if !ok {
		return solutionPlan{}, fmt.Errorf("%w: %T", errWrongFlags, flags)
	}

	if !prj.SQL {
		return solutionPlan{}, errSolutionNeedsSQL
	}

	cookie, err := resolveSession(f.session, pr)
	if err != nil {
		return solutionPlan{}, err
	}

	spec := userSpec{
		Module:            prj.Module,
		RizottoModule:     rizottoModule,
		Cookie:            cookie,
		SessionCookieName: "session",
		Methods:           userMethods(),
	}

	return solutionPlan{
		Data:         spec,
		Files:        spec.files(),
		Services:     nil,
		Controllers:  nil,
		Repositories: []sqlcEntry{newSqlcEntry(spec.Dir(), "user-queries.sql")},
		Env:          spec.env(),
		Steps:        spec.steps(),
	}, nil
}

// resolveSession reads the -session flag and falls back to asking the author.
func resolveSession(flagValue string, pr *prompter) (bool, error) {
	answer, err := answer(pr, flagValue,
		"Session in a cookie or a bearer header (cookie, bearer)", sessionCookie, normalizeSession)
	if err != nil {
		return false, err
	}

	return answer == sessionCookie, nil
}

func normalizeSession(raw string) (string, error) {
	answer := strings.ToLower(strings.TrimSpace(raw))
	if answer == sessionCookie || answer == sessionBearer {
		return answer, nil
	}

	return "", fmt.Errorf("%w, got %q", errSessionAnswer, raw)
}

func (u userSpec) files() []fileSpec {
	dir := u.Dir()

	return []fileSpec{
		{tmpl: "solutions/user/client.go.tmpl", out: dir + "/user-client/client.go"},
		{tmpl: "solutions/user/bind-gen.go.tmpl", out: dir + "/user-client/bind-gen.go"},
		{tmpl: "solutions/user/user-service.go.tmpl", out: dir + "/user-service.go"},
		{tmpl: "solutions/user/repository.go.tmpl", out: dir + "/repository/user-repository.go"},
		{tmpl: "solutions/user/schema.sql.tmpl", out: dir + "/repository/sql/schema.sql"},
		{tmpl: "solutions/user/queries.sql.tmpl", out: dir + "/repository/sql/user-queries.sql"},
		{tmpl: "solutions/user/migration.sql.tmpl", out: dir + "/repository/migrations/000001_create_users.sql"},
		{tmpl: "solutions/user/db/db.go.tmpl", out: dir + "/repository/db/db.go"},
		{tmpl: "solutions/user/db/models.go.tmpl", out: dir + "/repository/db/models.go"},
		{tmpl: "solutions/user/db/queries.sql.go.tmpl", out: dir + "/repository/db/user-queries.sql.go"},
		{tmpl: "solutions/user/auth-area.go.tmpl", out: "api/auth-area.go"},
		{tmpl: "solutions/user/user-api.go.tmpl", out: "api/user-api/user.go"},
		{tmpl: "solutions/user/user-api_test.go.tmpl", out: "api/user-api/user_test.go"},
	}
}

func (u userSpec) env() []envVar {
	return []envVar{
		{
			Key:     "APP_LOGIN_REDIRECT_URL",
			Value:   "http://localhost:3000",
			Comment: "where the caller lands once the session exists",
		},
		{
			Key:     "SESSION_TTL_HOURS",
			Value:   "720",
			Comment: "how long a session stays valid",
		},
	}
}

func (u userSpec) steps() []string {
	register := fmt.Sprintf(`register the service in server.go, before the controllers:

         import (
             "%s"
             usersrv "%s/%s"
         )

         user.RegisterServer(usersrv.NewUserService(pgPool))`,
		u.ClientImport(), u.Module, u.Dir())

	routes := `join the route table in server.go:

         import "` + u.Module + `/api/user-api"

         gt.Routing(
             gateway.JoinRouteTables(
                 userapi.NewUserController().RouteTable(),
             ),
         )`

	area := `authenticate the private routes: api.AuthArea is the real area function,
     so replace api.PrivateArea with api.AuthArea and api.UserHTTPContext with
     api.AuthHTTPContext in the controllers that need the current user.
     api/controller.go keeps its placeholder for the rest.`

	return []string{
		stepEnvExample,
		register,
		routes,
		area,
		stepGen,
		"task migrate-up # applies the tables of the solution",
		"task test       # runs the route tests and writes api/user-api/openapi.yml",
	}
}
