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

//go:embed templates
var templatesFS embed.FS

const (
	dirPerm  os.FileMode = 0o750
	filePerm os.FileMode = 0o644

	// sampleService is the service make-project scaffolds so that a fresh project
	// has a working example to copy.
	sampleService = "Item"
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
	{tmpl: "golangci.yml.tmpl", out: ".golangci.yml", sqlOnly: always},
	{tmpl: "env.tmpl", out: ".env", sqlOnly: always},
	{tmpl: "env.example.tmpl", out: ".env.example", sqlOnly: always},
	{tmpl: "go.mod.tmpl", out: "go.mod", sqlOnly: always},
	{tmpl: "Taskfile.yml.tmpl", out: "Taskfile.yml", sqlOnly: always},
	{tmpl: "README.md.tmpl", out: "README.md", sqlOnly: always},
	{tmpl: "server.go.tmpl", out: "server.go", sqlOnly: always},

	{tmpl: "api/controller.go.tmpl", out: "api/controller.go", sqlOnly: always},
	{tmpl: "api/doc/main.go.tmpl", out: "api/doc/main.go", sqlOnly: always},
}

// generateProject writes the project skeleton and its sample service.
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

	service, err := generateService(spec, target)
	if err != nil {
		return nil, err
	}

	return append(written, service...), nil
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
