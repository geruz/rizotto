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
// directories, so that templates/.agents ships with the binary.
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
	// sqlOnly keeps the file out of projects generated without SQL.
	sqlOnly bool
}

// always and sqlOnly name the two kinds of entries in the manifest below.
const (
	always  = false
	sqlOnly = true
)

//nolint:gochecknoglobals // the scaffold manifest is static
var projectFiles = []fileSpec{
	{tmpl: "gitignore.tmpl", out: ".gitignore", sqlOnly: always},
	{tmpl: "Taskfile.yml.tmpl", out: "Taskfile.yml", sqlOnly: always},
	{tmpl: "README.md.tmpl", out: "README.md", sqlOnly: always},
	{tmpl: ".agents/skills/rizotto/SKILL.md.tmpl", out: ".agents/skills/rizotto/SKILL.md", sqlOnly: always},

	{tmpl: "server/golangci.yml.tmpl", out: "server/.golangci.yml", sqlOnly: always},
	{tmpl: "server/env.tmpl", out: "server/.env", sqlOnly: always},
	{tmpl: "server/env.example.tmpl", out: "server/.env.example", sqlOnly: always},
	{tmpl: "server/go.mod.tmpl", out: "server/go.mod", sqlOnly: always},
	{tmpl: "server/Taskfile.yml.tmpl", out: "server/Taskfile.yml", sqlOnly: always},
	{tmpl: "server/server.go.tmpl", out: "server/server.go", sqlOnly: always},
	{tmpl: "server/api/controller.go.tmpl", out: "server/api/controller.go", sqlOnly: always},
	{tmpl: "server/api/doc/main.go.tmpl", out: "server/api/doc/main.go", sqlOnly: always},

	{tmpl: "web/package.json.tmpl", out: "web/package.json", sqlOnly: always},
	{tmpl: "web/Taskfile.yml.tmpl", out: "web/Taskfile.yml", sqlOnly: always},
	{tmpl: "web/components.json.tmpl", out: "web/components.json", sqlOnly: always},
	{tmpl: "web/tsconfig.json.tmpl", out: "web/tsconfig.json", sqlOnly: always},
	{tmpl: "web/tsconfig.app.json.tmpl", out: "web/tsconfig.app.json", sqlOnly: always},
	{tmpl: "web/tsconfig.node.json.tmpl", out: "web/tsconfig.node.json", sqlOnly: always},
	{tmpl: "web/vite.config.ts.tmpl", out: "web/vite.config.ts", sqlOnly: always},
	{tmpl: "web/index.html.tmpl", out: "web/index.html", sqlOnly: always},
	{tmpl: "web/src/index.css.tmpl", out: "web/src/index.css", sqlOnly: always},
	{tmpl: "web/src/main.tsx.tmpl", out: "web/src/main.tsx", sqlOnly: always},
	{tmpl: "web/src/App.tsx.tmpl", out: "web/src/App.tsx", sqlOnly: always},
	{tmpl: "web/src/lib/utils.ts.tmpl", out: "web/src/lib/utils.ts", sqlOnly: always},
	{tmpl: "web/src/components/ui/button.tsx.tmpl", out: "web/src/components/ui/button.tsx", sqlOnly: always},
	{tmpl: "web/src/components/ui/card.tsx.tmpl", out: "web/src/components/ui/card.tsx", sqlOnly: always},
}

// generateProject writes the project skeleton: the go module in server/ with its
// sample service, and the React application in web/.
func generateProject(prj project, target string) ([]string, error) {
	files := make([]fileSpec, 0, len(projectFiles))

	for _, spec := range projectFiles {
		if spec.sqlOnly && !prj.SQL {
			continue
		}

		files = append(files, spec)
	}

	written, err := generate(prj, files, target)
	if err != nil {
		return nil, err
	}

	spec, err := newServiceSpec(prj.Module, sampleService, methodKeys(), prj.SQL)
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
