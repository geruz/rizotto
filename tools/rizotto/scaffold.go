package main

import (
	"bytes"
	"embed"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// templatesFS holds the project templates. The all: prefix keeps the dot
// directories, so that templates/.agents and templates/.docker ship with the
// binary.
//
//go:embed all:templates
var templatesFS embed.FS

const (
	dirPerm  os.FileMode = 0o750
	filePerm os.FileMode = 0o644

	// sampleService is the service make-project scaffolds so that a fresh project
	// has a working example to copy.
	sampleService = "Item"

	// serverDir holds the go module of a generated project, webDir its React
	// application. Every other command works inside serverDir.
	serverDir = "server"
	webDir    = "web"
)

// fileSpec maps one embedded template onto one file of the generated project.
type fileSpec struct {
	// tmpl is the path inside the embedded templates directory.
	tmpl string
	// out is the path of the rendered file, relative to the project root.
	out string
}

//nolint:gochecknoglobals // the scaffold manifest is static
var projectFiles = []fileSpec{
	{tmpl: "gitignore.tmpl", out: ".gitignore"},
	{tmpl: "Taskfile.yml.tmpl", out: "Taskfile.yml"},
	{tmpl: "README.md.tmpl", out: "README.md"},
	{tmpl: ".agents/skills/rizotto/SKILL.md.tmpl", out: ".agents/skills/rizotto/SKILL.md"},
	{tmpl: ".agents/skills/database/SKILL.md.tmpl", out: ".agents/skills/database/SKILL.md"},
	{tmpl: ".docker/Dockerfile.tmpl", out: ".docker/Dockerfile"},
	{tmpl: ".docker/Dockerfile.dockerignore.tmpl", out: ".docker/Dockerfile.dockerignore"},
	{tmpl: ".docker/env.tmpl", out: ".docker/env"},
	{tmpl: ".docker/migrations.Dockerfile.tmpl", out: ".docker/migrations.Dockerfile"},
	{tmpl: ".docker/migrations.Dockerfile.dockerignore.tmpl", out: ".docker/migrations.Dockerfile.dockerignore"},

	{tmpl: "server/golangci.yml.tmpl", out: "server/.golangci.yml"},
	{tmpl: "server/env.tmpl", out: "server/.env"},
	{tmpl: "server/env.example.tmpl", out: "server/.env.example"},
	{tmpl: "server/go.mod.tmpl", out: "server/go.mod"},
	{tmpl: "server/Taskfile.yml.tmpl", out: "server/Taskfile.yml"},
	{tmpl: "server/docker-compose.yml.tmpl", out: "server/docker-compose.yml"},
	{tmpl: "server/scripts/migrate.sh.tmpl", out: "server/scripts/migrate.sh"},
	{tmpl: "server/server.go.tmpl", out: "server/server.go"},
	{tmpl: "server/api/controller.go.tmpl", out: "server/api/controller.go"},
	{tmpl: "server/api/doc/main.go.tmpl", out: "server/api/doc/main.go"},

	{tmpl: "web/package.json.tmpl", out: "web/package.json"},
	{tmpl: "web/Taskfile.yml.tmpl", out: "web/Taskfile.yml"},
	{tmpl: "web/components.json.tmpl", out: "web/components.json"},
	{tmpl: "web/tsconfig.json.tmpl", out: "web/tsconfig.json"},
	{tmpl: "web/tsconfig.app.json.tmpl", out: "web/tsconfig.app.json"},
	{tmpl: "web/tsconfig.node.json.tmpl", out: "web/tsconfig.node.json"},
	{tmpl: "web/vite.config.ts.tmpl", out: "web/vite.config.ts"},
	{tmpl: "web/index.html.tmpl", out: "web/index.html"},
	{tmpl: "web/src/index.css.tmpl", out: "web/src/index.css"},
	{tmpl: "web/src/main.tsx.tmpl", out: "web/src/main.tsx"},
	{tmpl: "web/src/App.tsx.tmpl", out: "web/src/App.tsx"},
	{tmpl: "web/src/lib/utils.ts.tmpl", out: "web/src/lib/utils.ts"},
	{tmpl: "web/src/components/ui/button.tsx.tmpl", out: "web/src/components/ui/button.tsx"},
	{tmpl: "web/src/components/ui/card.tsx.tmpl", out: "web/src/components/ui/card.tsx"},
}

// generateProject writes the project skeleton: the go module in server/ with its
// sample service, and the React application in web/.
func generateProject(prj project, target string) ([]string, error) {
	written, err := generate(prj, projectFiles, target)
	if err != nil {
		return nil, err
	}

	// The sample service owns a table, so that a fresh project shows the whole
	// path from the controller down to the migrations.
	const sampleServiceDB = true

	spec, err := newServiceSpec(prj.Module, sampleService, methodKeys(), sampleServiceDB)
	if err != nil {
		return nil, err
	}

	service, err := generateService(spec, filepath.Join(target, serverDir))
	if err != nil {
		return nil, err
	}

	for _, file := range service {
		written = append(written, serverDir+"/"+file)
	}

	return written, nil
}

// generate renders the given files with data into target and returns what it wrote.
func generate(data any, files []fileSpec, target string) ([]string, error) {
	written := make([]string, 0, len(files))

	for _, spec := range files {
		content, err := render(spec, data)
		if err != nil {
			return nil, err
		}

		path := filepath.Join(target, spec.out)

		err = os.MkdirAll(filepath.Dir(path), dirPerm)
		if err != nil {
			return nil, fmt.Errorf("create directory for %s: %w", spec.out, err)
		}

		err = os.WriteFile(path, content, filePerm)
		if err != nil {
			return nil, fmt.Errorf("write %s: %w", spec.out, err)
		}

		written = append(written, spec.out)
	}

	return written, nil
}

func render(spec fileSpec, data any) ([]byte, error) {
	raw, err := templatesFS.ReadFile("templates/" + spec.tmpl)
	if err != nil {
		return nil, fmt.Errorf("read template %s: %w", spec.tmpl, err)
	}

	tmpl, err := template.New(spec.tmpl).Option("missingkey=error").Parse(string(raw))
	if err != nil {
		return nil, fmt.Errorf("parse template %s: %w", spec.tmpl, err)
	}

	var buf bytes.Buffer

	err = tmpl.Execute(&buf, data)
	if err != nil {
		return nil, fmt.Errorf("render template %s: %w", spec.tmpl, err)
	}

	if !strings.HasSuffix(spec.out, ".go") {
		return buf.Bytes(), nil
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("format generated %s: %w", spec.out, err)
	}

	return formatted, nil
}
