package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shoenig/test/must"
)

const (
	providersFlag = "-providers"
	sessionFlag   = "-session"
	downloadFlag  = "-download"
	forceFlag     = "-force"
)

// newSQLProjectDir creates a project the solutions requiring a database accept:
// findProject needs the go.mod, inspectProject reads the SQL support back from the
// presence of sqlc.yaml.
func newSQLProjectDir(t *testing.T) string {
	t.Helper()

	root := newProjectDir(t)

	err := os.WriteFile(filepath.Join(root, sqlcConfigFile), []byte("version: \"2\"\nsql: []\n"), filePerm)
	must.NoError(t, err)

	return root
}

// addSolutionArgs is one non-interactive installation.
func addSolutionArgs(name, root string, rest ...string) []string {
	const fixed = 5

	args := make([]string, 0, fixed+len(rest))
	args = append(args, addCommand, name, projectFlag, root, skipTidyFlag)

	return append(args, rest...)
}

func Test_runSolution_ListsWhatCanBeInstalled(t *testing.T) {
	t.Parallel()

	out := &bytes.Buffer{}

	err := runSolution([]string{listCommand}, strings.NewReader(""), out)
	must.NoError(t, err)

	printed := out.String()
	must.StrContains(t, printed, userSolutionName)
	must.StrContains(t, printed, oauthSolutionName)
}

func Test_runSolution_AsksEveryStep(t *testing.T) {
	t.Parallel()

	root := newSQLProjectDir(t)
	out := &bytes.Buffer{}

	err := runSolution(
		[]string{addCommand, userSolutionName, projectFlag, root, skipTidyFlag},
		strings.NewReader(sessionCookie+"\n"), out)
	must.NoError(t, err)

	must.StrContains(t, out.String(), "Session in a cookie or a bearer header")
}

func Test_runSolutionAdd_WritesTheUserVertical(t *testing.T) {
	t.Parallel()

	root := newSQLProjectDir(t)
	out := &bytes.Buffer{}

	err := runSolution(addSolutionArgs(userSolutionName, root, sessionFlag, sessionCookie), strings.NewReader(""), out)
	must.NoError(t, err)

	mustHaveFiles(t, root, written(out.String()),
		"services/user/user-client/client.go",
		"services/user/user-client/bind-gen.go",
		"services/user/user-service.go",
		"services/user/repository/user-repository.go",
		"services/user/repository/sql/schema.sql",
		"services/user/repository/sql/user-queries.sql",
		"services/user/repository/migrations/000001_create_users.sql",
		"services/user/repository/db/db.go",
		"services/user/repository/db/models.go",
		"services/user/repository/db/user-queries.sql.go",
		"api/auth-area.go",
		"api/user-api/user.go",
		"api/user-api/user_test.go",
		sqlcConfigFile,
	)

	// the user solution knows nothing about logging somebody in
	must.DirNotExists(t, filepath.Join(root, "services", "identity"))
	must.FileNotExists(t, filepath.Join(root, "api", "oauth-api", "oauth.go"))
}

// Test_runSolutionAdd_InstallsWhatItRequires pins the dependency mechanism: oauth
// is built on user, so asking for oauth alone installs both.
func Test_runSolutionAdd_InstallsWhatItRequires(t *testing.T) {
	t.Parallel()

	root := newSQLProjectDir(t)
	out := &bytes.Buffer{}

	err := runSolution(
		addSolutionArgs(oauthSolutionName, root, providersFlag, providerGoogle), strings.NewReader(""), out)
	must.NoError(t, err)

	printed := out.String()
	must.StrContains(t, printed, `"oauth" requires "user"`)
	must.StrContains(t, printed, `Added solution "user"`)
	must.StrContains(t, printed, `Added solution "oauth"`)

	files := written(printed)
	mustHaveFiles(t, root, files,
		"services/user/user-service.go",
		"api/auth-area.go",
		"services/identity/identity-service.go",
		"services/identity/identity-client/client.go",
		"services/identity/repository/identity-repository.go",
		"services/identity/repository/migrations/000001_create_user_identities.sql",
		"services/identity/oauth/provider.go",
		"api/oauth-api/oauth.go",
		"api/oauth-api/oauth_test.go",
	)
}

// Test_runSolutionAdd_KeepsAnAlreadyInstalledDependency pins that installing the
// dependency by hand first, with answers of its own, is not undone by the
// solution requiring it.
func Test_runSolutionAdd_KeepsAnAlreadyInstalledDependency(t *testing.T) {
	t.Parallel()

	root := newSQLProjectDir(t)
	out := &bytes.Buffer{}

	err := runSolution(addSolutionArgs(userSolutionName, root, sessionFlag, sessionBearer), strings.NewReader(""), out)
	must.NoError(t, err)

	area := readFile(t, root, "api/auth-area.go")
	must.StrContains(t, area, "Bearer ")

	out = &bytes.Buffer{}
	err = runSolution(
		addSolutionArgs(oauthSolutionName, root, providersFlag, providerGoogle), strings.NewReader(""), out)
	must.NoError(t, err)

	printed := out.String()
	must.StrNotContains(t, printed, `requires "user"`)
	must.StrNotContains(t, printed, `Added solution "user"`)

	// the bearer answer given to the user solution survives
	must.Eq(t, area, readFile(t, root, "api/auth-area.go"))
}

func Test_runSolutionAdd_WritesTheFilesVertical(t *testing.T) {
	t.Parallel()

	root := newSQLProjectDir(t)
	out := &bytes.Buffer{}

	err := runSolution(
		addSolutionArgs(filesSolutionName, root, downloadFlag, downloadRedirect), strings.NewReader(""), out)
	must.NoError(t, err)

	mustHaveFiles(t, root, written(out.String()),
		"services/file/file-client/client.go",
		"services/file/file-client/bind-gen.go",
		"services/file/file-service.go",
		"services/file/storage/s3.go",
		"services/file/storage/s3_test.go",
		"services/file/repository/file-repository.go",
		"services/file/repository/sql/schema.sql",
		"services/file/repository/sql/file-queries.sql",
		"services/file/repository/migrations/000001_create_files.sql",
		"services/file/repository/db/db.go",
		"services/file/repository/db/models.go",
		"services/file/repository/db/file-queries.sql.go",
		"api/file-api/file.go",
		"api/file-api/file_test.go",
		// pulled in as a dependency
		"services/user/user-service.go",
		"api/auth-area.go",
	)

	printed := out.String()
	must.StrContains(t, printed, `"files" requires "user"`)
	must.StrContains(t, printed, "S3_BUCKET")
	must.StrContains(t, printed, "file.RegisterServer")
}

// Test_runSolutionAdd_FilesDownloadVariants pins that the flag reaches the
// generated code rather than only the report.
func Test_runSolutionAdd_FilesDownloadVariants(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		downloadRedirect: "gateway.Redirect(\"GET /api/v1/files/{file_id}/content\"",
		downloadURL:      "gateway.JSONMethod(\"GET /api/v1/files/{file_id}/content\"",
	}

	for download, expected := range cases {
		t.Run(download, func(t *testing.T) {
			t.Parallel()

			root := newSQLProjectDir(t)
			out := &bytes.Buffer{}

			err := runSolution(
				addSolutionArgs(filesSolutionName, root, downloadFlag, download), strings.NewReader(""), out)
			must.NoError(t, err)
			must.StrContains(t, readFile(t, root, "api/file-api/file.go"), expected)
		})
	}
}

func Test_runSolutionAdd_RegistersItsQueriesWithSqlc(t *testing.T) {
	t.Parallel()

	root := newSQLProjectDir(t)
	out := &bytes.Buffer{}

	err := runSolution(
		addSolutionArgs(oauthSolutionName, root, providersFlag, providerGoogle), strings.NewReader(""), out)
	must.NoError(t, err)

	config := readFile(t, root, sqlcConfigFile)
	must.StrContains(t, config, "./services/user/repository/sql/user-queries.sql")
	must.StrContains(t, config, "./services/identity/repository/sql/identity-queries.sql")
	must.StrContains(t, config, "./services/identity/repository/db")
}

// Test_runSolutionAdd_RefusesToOverwrite pins that installing twice does not
// silently replace files the project may have edited since.
func Test_runSolutionAdd_RefusesToOverwrite(t *testing.T) {
	t.Parallel()

	root := newSQLProjectDir(t)
	out := &bytes.Buffer{}
	args := addSolutionArgs(userSolutionName, root, sessionFlag, sessionCookie)

	err := runSolution(args, strings.NewReader(""), out)
	must.NoError(t, err)

	err = runSolution(args, strings.NewReader(""), out)
	must.ErrorIs(t, err, errSolutionConflict)

	err = runSolution(append(args, forceFlag), strings.NewReader(""), out)
	must.NoError(t, err)
}

// Test_runSolutionAdd_RefusesAProjectWithoutSQL pins that a solution owning tables
// says so instead of writing a repository the project cannot generate.
func Test_runSolutionAdd_RefusesAProjectWithoutSQL(t *testing.T) {
	t.Parallel()

	out := &bytes.Buffer{}

	err := runSolution(
		addSolutionArgs(userSolutionName, newProjectDir(t), sessionFlag, sessionCookie), strings.NewReader(""), out)
	must.ErrorIs(t, err, errSolutionNeedsSQL)
}

func Test_runSolutionAdd_ReportsWhatHasToBeWiredByHand(t *testing.T) {
	t.Parallel()

	root := newSQLProjectDir(t)
	out := &bytes.Buffer{}

	err := runSolution(
		addSolutionArgs(oauthSolutionName, root, providersFlag, providerGoogle), strings.NewReader(""), out)
	must.NoError(t, err)

	printed := out.String()
	// from the user solution
	must.StrContains(t, printed, "SESSION_TTL_HOURS")
	must.StrContains(t, printed, "user.RegisterServer")
	must.StrContains(t, printed, "api.AuthArea")
	// from the oauth solution
	must.StrContains(t, printed, "OAUTH_GOOGLE_CLIENT_ID")
	must.StrContains(t, printed, "identity.RegisterServer")
	must.StrContains(t, printed, "/api/v1/auth/<provider>/callback")
}

func Test_runSolution_RejectsBadArguments(t *testing.T) {
	t.Parallel()

	root := newSQLProjectDir(t)
	cases := map[string][]string{
		caseMissingSubcommand: {},
		unknownSubcommand:     {"remove"},
		"missing name":        {addCommand, projectFlag, root},
		"unknown solution":    {addCommand, "sorcery", projectFlag, root},
		"unknown provider":    {addCommand, oauthSolutionName, projectFlag, root, providersFlag, "myspace"},
		"unknown session":     {addCommand, userSolutionName, projectFlag, root, sessionFlag, "telepathy"},
		casePositional:        {addCommand, userSolutionName, "extra", projectFlag, root},
		"list with arguments": {listCommand, userSolutionName},
		"skill of nothing":    {skillCommand, "sorcery"},
	}

	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			out := &bytes.Buffer{}

			err := runSolution(append(args, skipTidyFlag), strings.NewReader(""), out)
			must.ErrorIs(t, err, errUsage)
		})
	}
}

func Test_runSolution_PrintsSkills(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"":                "# Solutions",
		userSolutionName:  "# Users",
		oauthSolutionName: "# OAuth",
		filesSolutionName: "# Files",
	}

	for name, heading := range cases {
		args := []string{skillCommand}
		if name != "" {
			args = append(args, name)
		}

		out := &bytes.Buffer{}
		err := runSolution(args, strings.NewReader(""), out)
		must.NoError(t, err)
		must.StrContains(t, out.String(), heading)
	}
}

// Test_installOrder_PutsDependenciesFirst pins the order the runner installs in,
// independently of the command that drove it.
func Test_installOrder_PutsDependenciesFirst(t *testing.T) {
	t.Parallel()

	prj := projectInfo{Root: newSQLProjectDir(t), Module: testModule, SQL: true, Services: nil}

	order, err := installOrder(oauthSolution{}, prj)
	must.NoError(t, err)
	must.SliceLen(t, 2, order)
	must.Eq(t, userSolutionName, order[0].Name())
	must.Eq(t, oauthSolutionName, order[1].Name())

	order, err = installOrder(userSolution{}, prj)
	must.NoError(t, err)
	must.SliceLen(t, 1, order)
	must.Eq(t, userSolutionName, order[0].Name())

	order, err = installOrder(filesSolution{}, prj)
	must.NoError(t, err)
	must.SliceLen(t, 2, order)
	must.Eq(t, userSolutionName, order[0].Name())
	must.Eq(t, filesSolutionName, order[1].Name())
}

// Test_Requires_NamesKnownSolutions pins that every requirement in the registry
// resolves, so a typo cannot ship as a runtime failure.
func Test_Requires_NamesKnownSolutions(t *testing.T) {
	t.Parallel()

	prj := projectInfo{Root: newSQLProjectDir(t), Module: testModule, SQL: true, Services: nil}

	for _, sol := range solutions {
		for _, name := range sol.Requires() {
			_, err := findSolution(name)
			must.NoError(t, err, must.Sprintf("%s requires %q", sol.Name(), name))
		}

		_, err := installOrder(sol, prj)
		must.NoError(t, err, must.Sprintf("the requirements of %s must resolve", sol.Name()))
	}
}

func Test_normalizeProviders(t *testing.T) {
	t.Parallel()

	both := providerGoogle + "," + providerGitHub
	cases := map[string]string{
		providerGoogle:                        providerGoogle,
		"gh":                                  providerGitHub,
		providerGitHub + "," + providerGoogle: both,
		providerGoogle + ", gh":               both,
		"GOOGLE":                              providerGoogle,
	}

	for answer, expected := range cases {
		got, err := normalizeProviders(answer)
		must.NoError(t, err)
		must.Eq(t, expected, got)
	}

	_, err := normalizeProviders("myspace")
	must.ErrorIs(t, err, errUnknownProvider)

	_, err = normalizeProviders("")
	must.ErrorIs(t, err, errNoProviders)
}

// written turns the report back into the list of files it printed, so the tests
// can check what was reported as well as what landed on disk.
func written(report string) []string {
	files := []string{}

	for line := range strings.SplitSeq(report, "\n") {
		if !strings.HasPrefix(line, "  ") {
			continue
		}

		file := strings.TrimSpace(line)
		if strings.Contains(file, " ") || strings.Contains(file, "=") {
			continue
		}

		files = append(files, file)
	}

	return files
}
