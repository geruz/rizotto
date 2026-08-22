package main

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shoenig/test/must"
)

const (
	testName = "MyShop"
	testRepo = "github.com/acme/shop"
)

func Test_generate_ProjectWithoutSQL(t *testing.T) {
	t.Parallel()

	target := t.TempDir()
	prj := mustNewProject(t, testName, false)

	files, err := generateProject(prj, target)
	must.NoError(t, err)

	mustHaveFiles(t, target, files,
		".claude/skills/rizotto/SKILL.md",
		".env",
		".gitignore",
		".golangci.yml",
		"README.md",
		"Taskfile.yml",
		"api/controller.go",
		"api/doc/main.go",
		"go.mod",
		"server.go",
		"services/item/item-client/bind-gen.go",
		"services/item/item-client/client.go",
		"services/item/item-service.go",
	)

	for _, notGenerated := range []string{"sqlc.yaml", "services/item/repository/db/db.go"} {
		must.SliceNotContains(t, files, notGenerated)
	}

	must.StrContains(t, readFile(t, target, "go.mod"), "module "+testRepo+"\n")
	must.StrNotContains(t, readFile(t, target, ".env"), "DATABASE_URL")
	must.StrContains(t, readFile(t, target, ".env.example"), "SERVICE_NAME=myshop")
	must.StrContains(t, readFile(t, target, "server.go"), "itemsrv.NewItemService()")
	must.StrContains(t, readFile(t, target, "services/item/item-service.go"), "[]item.Item")
	must.StrContains(t, readFile(t, target, "services/item/item-client/client.go"), "http://item-service/item/get")

	skill := readFile(t, target, ".claude/skills/rizotto/SKILL.md")
	must.StrContains(t, skill, "name: rizotto")
	must.StrContains(t, skill, "rizotto skill")
}

func Test_generate_ProjectWithSQL(t *testing.T) {
	t.Parallel()

	target := t.TempDir()
	prj := mustNewProject(t, testName, true)

	files, err := generateProject(prj, target)
	must.NoError(t, err)

	mustHaveFiles(t, target, files,
		"sqlc.yaml",
		"services/item/repository/db/db.go",
		"services/item/repository/db/item-queries.sql.go",
		"services/item/repository/db/models.go",
		"services/item/repository/item-repository.go",
		"services/item/repository/migrations/000001_create_items.sql",
		"services/item/repository/sql/item-queries.sql",
		"services/item/repository/sql/schema.sql",
	)

	must.StrContains(t, readFile(t, target, ".env"), "postgres://postgres:postgres@localhost:5432/myshop")
	must.StrContains(t, readFile(t, target, "server.go"), "pg.MustOpenConnection")
	must.StrContains(t, readFile(t, target, "Taskfile.yml"), "{{.SQLC_VERSION}}")
	must.StrContains(t, readFile(t, target, "services/item/item-service.go"), "repository.NewItemRepository")
}

func Test_generate_NoControllerIsScaffolded(t *testing.T) {
	t.Parallel()

	target := t.TempDir()

	files, err := generateProject(mustNewProject(t, testName, true), target)
	must.NoError(t, err)

	for _, file := range files {
		must.False(t, strings.Contains(file, "item-api"), must.Sprintf("unexpected controller file %s", file))
	}

	must.StrNotContains(t, readFile(t, target, "server.go"), "NewItemController")
}

func Test_generate_ImportsFollowTheRepository(t *testing.T) {
	t.Parallel()

	target := t.TempDir()

	prj, err := newProject(testName, "git@gitlab.com:acme/shop.git", "1.25.0", "..", true)
	must.NoError(t, err)

	_, err = generateProject(prj, target)
	must.NoError(t, err)

	gomod := readFile(t, target, "go.mod")
	must.StrContains(t, gomod, "module gitlab.com/acme/shop\n")
	must.StrContains(t, gomod, "go 1.25.0\n")
	must.StrContains(t, gomod, "replace "+rizottoModule+" => ")
	must.StrContains(t, readFile(t, target, "server.go"), `itemsrv "gitlab.com/acme/shop/services/item"`)
	must.StrContains(t, readFile(t, target, "services/item/item-service.go"),
		`"gitlab.com/acme/shop/services/item/repository"`)
}

func mustNewProject(t *testing.T, name string, withSQL bool) project {
	t.Helper()

	prj, err := newProject(name, testRepo, "1.25.0", "", withSQL)
	must.NoError(t, err)

	return prj
}

// mustHaveFiles checks that every expected path was reported and written, and that
// the generated go files parse.
func mustHaveFiles(t *testing.T, target string, written []string, expected ...string) {
	t.Helper()

	for _, path := range expected {
		must.SliceContains(t, written, path)
		must.FileExists(t, filepath.Join(target, path))
	}

	for _, path := range written {
		if !strings.HasSuffix(path, ".go") {
			continue
		}

		_, err := parser.ParseFile(token.NewFileSet(), filepath.Join(target, path), nil, parser.AllErrors)
		must.NoError(t, err, must.Sprintf("generated %s does not parse", path))
	}
}

func readFile(t *testing.T, target string, path string) string {
	t.Helper()

	content, err := os.ReadFile(filepath.Join(target, path)) //nolint:gosec // test reads its own temp dir
	must.NoError(t, err)

	return string(content)
}
