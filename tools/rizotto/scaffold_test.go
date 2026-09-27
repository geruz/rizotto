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

func Test_generate_Project(t *testing.T) {
	t.Parallel()

	target := t.TempDir()
	prj := mustNewProject(t, testName)

	files, err := generateProject(prj, target)
	must.NoError(t, err)

	mustHaveFiles(t, target, files,
		".agents/skills/rizotto/SKILL.md",
		".docker/Dockerfile",
		".docker/Dockerfile.dockerignore",
		".docker/env",
		".gitignore",
		"README.md",
		"Taskfile.yml",
		"server/.env",
		"server/.golangci.yml",
		"server/Taskfile.yml",
		"server/api/controller.go",
		"server/api/doc/main.go",
		"server/go.mod",
		"server/server.go",
		"server/services/item/item-client/bind-gen.go",
		"server/services/item/item-client/client.go",
		"server/services/item/item-service.go",
		"web/Taskfile.yml",
		"web/components.json",
		"web/package.json",
		"web/src/App.tsx",
		"web/src/components/ui/button.tsx",
		"web/src/lib/utils.ts",
		"web/vite.config.ts",
	)

	must.StrContains(t, readFile(t, target, "server/go.mod"), "module "+testRepo+"/server\n")
	must.StrContains(t, readFile(t, target, "server/.env.example"), "SERVICE_NAME=myshop")
	must.StrContains(t, readFile(t, target, "server/services/item/item-service.go"), "[]item.Item")
	must.StrContains(t, readFile(t, target, "server/services/item/item-client/client.go"),
		"http://item-service/item/get")

	must.StrContains(t, readFile(t, target, "web/package.json"), `"name": "myshop-web"`)
	must.StrContains(t, readFile(t, target, "web/vite.config.ts"), `"/api": "http://localhost:9090"`)
	must.StrContains(t, readFile(t, target, "web/src/App.tsx"), "<CardTitle>MyShop</CardTitle>")
	must.StrContains(t, readFile(t, target, "Taskfile.yml"), "docker build -f .docker/Dockerfile -t myshop .")
	must.StrContains(t, readFile(t, target, ".docker/Dockerfile"), "COPY --from=web /src/web/dist ./static")
	must.StrContains(t, readFile(t, target, ".docker/env"), "STATIC_DIR=/app/static")
	must.StrContains(t, readFile(t, target, "server/server.go"), `gt.Static("/", staticDir)`)

	skill := readFile(t, target, ".agents/skills/rizotto/SKILL.md")
	must.StrContains(t, skill, "name: rizotto")
	must.StrContains(t, skill, "rizotto skill")
}

func Test_generate_ProjectWithDatabase(t *testing.T) {
	t.Parallel()

	target := t.TempDir()

	files, err := generateProject(mustNewProject(t, testName), target)
	must.NoError(t, err)

	mustHaveFiles(t, target, files,
		".agents/skills/database/SKILL.md",
		".docker/migrations.Dockerfile",
		".docker/migrations.Dockerfile.dockerignore",
		"server/docker-compose.yml",
		"server/scripts/migrate.sh",
		"server/sqlc.yaml",
		"server/services/item/repository/db/db.go",
		"server/services/item/repository/db/item-queries.sql.go",
		"server/services/item/repository/db/models.go",
		"server/services/item/repository/item-repository.go",
		"server/services/item/repository/migrations/000001_create_items.sql",
		"server/services/item/repository/sql/item-queries.sql",
		"server/services/item/repository/sql/schema.sql",
	)

	must.StrContains(t, readFile(t, target, "server/.env"), "postgres://postgres:postgres@localhost:5432/myshop")
	must.StrContains(t, readFile(t, target, "server/server.go"), "pg.MustOpenConnection")
	must.StrContains(t, readFile(t, target, "server/services/item/item-service.go"), "repository.NewItemRepository")
	must.StrContains(t, readFile(t, target, "server/Taskfile.yml"), "{{.SQLC_VERSION}}")
	must.StrContains(t, readFile(t, target, "server/Taskfile.yml"), "docker compose run --rm")
	must.StrContains(t, readFile(t, target, "server/Taskfile.yml"), `ARGS: "redo {{.CLI_ARGS}}"`)
	must.StrContains(t, readFile(t, target, "Taskfile.yml"), "task -d server migrate-up")
	must.StrContains(t, readFile(t, target, "Taskfile.yml"), "task -d server migrate-new -- {{.CLI_ARGS}}")
	must.StrContains(t, readFile(t, target, "Taskfile.yml"),
		"docker build -f .docker/migrations.Dockerfile -t myshop-migrations .")

	compose := readFile(t, target, "server/docker-compose.yml")
	must.StrContains(t, compose, "name: myshop\n")
	must.StrContains(t, compose, "POSTGRES_DB: myshop")
	must.StrContains(t, compose, "postgres://postgres:postgres@postgres:5432/myshop")

	must.StrContains(t, readFile(t, target, "server/scripts/migrate.sh"), "public.dbmate_migrations_$target")
	must.StrContains(t, readFile(t, target, ".docker/migrations.Dockerfile"),
		`ENTRYPOINT ["sh", "/app/scripts/migrate.sh"]`)

	skill := readFile(t, target, ".agents/skills/database/SKILL.md")
	must.StrContains(t, skill, "name: database")
	must.StrContains(t, skill, "myshop-migrations")

	for _, file := range []string{"README.md", "Taskfile.yml", "server/Taskfile.yml", "server/.env"} {
		must.StrNotContains(t, readFile(t, target, file), "<no value>")
	}
}

func Test_generate_NoControllerIsScaffolded(t *testing.T) {
	t.Parallel()

	target := t.TempDir()

	files, err := generateProject(mustNewProject(t, testName), target)
	must.NoError(t, err)

	for _, file := range files {
		must.False(t, strings.Contains(file, "item-api"), must.Sprintf("unexpected controller file %s", file))
	}

	must.StrNotContains(t, readFile(t, target, "server/server.go"), "NewItemController")
}

func Test_generate_ImportsFollowTheRepository(t *testing.T) {
	t.Parallel()

	target := t.TempDir()

	prj, err := newProject(testName, "git@gitlab.com:acme/shop.git", "1.25.0", "..")
	must.NoError(t, err)

	_, err = generateProject(prj, target)
	must.NoError(t, err)

	gomod := readFile(t, target, "server/go.mod")
	must.StrContains(t, gomod, "module gitlab.com/acme/shop/server\n")
	must.StrContains(t, gomod, "go 1.25.0\n")
	must.StrContains(t, gomod, "replace "+rizottoModule+" => ")
	must.StrContains(t, readFile(t, target, "server/server.go"), `itemsrv "gitlab.com/acme/shop/server/services/item"`)
	must.StrContains(t, readFile(t, target, "server/services/item/item-service.go"),
		`"gitlab.com/acme/shop/server/services/item/repository"`)
}

func mustNewProject(t *testing.T, name string) project {
	t.Helper()

	prj, err := newProject(name, testRepo, "1.25.0", "")
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
