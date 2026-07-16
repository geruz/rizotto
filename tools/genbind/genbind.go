// nolint
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"strings"
	"text/template"
)

var (
	clientName = flag.String("client", "", "client struct")
)

type Field struct {
	Name string
	Type string
}
type Method struct {
	Name                string
	Request             Field
	Response            Field
	Address             string
	IsEvent             bool
	IsAuthTokenRequired bool
	TokenScope          string
}

type BindingGenData struct {
	Package                  string
	ServiceName              string
	ClientImplementationName string
	ClientName               string
	Source                   string
	Methods                  []Method
	MethodsForRegisterServer []Method
	IsAuthTokenRequired      bool
	IsTokenScopePresent      bool
}

func newBindingGenData(serviceName string) BindingGenData {
	clientName := strings.ReplaceAll(serviceName, "Service", "Client")
	return BindingGenData{
		ServiceName:              serviceName,
		ClientName:               clientName,
		ClientImplementationName: clientName + "Implementation",
		Package:                  os.Getenv("GOPACKAGE"),
		Source:                   os.Getenv("GOFILE"),
		Methods:                  []Method{},
		MethodsForRegisterServer: []Method{},
	}
}

func main() {

	flag.Parse()

	goFile := os.Getenv("GOFILE")
	fmt.Printf("Running generating  client for '%s' from %s\n", *clientName, goFile)

	fset := token.NewFileSet() // positions are relative to fset

	// get ast Node of whole file;
	ff, err := parser.ParseFile(fset, goFile, nil, parser.ParseComments)
	if err != nil {
		fmt.Println(err)
		return
	}
	ast.Inspect(ff, func(n ast.Node) bool {
		if d, ok := n.(*ast.GenDecl); ok {
			switch d.Tok {
			case token.TYPE:
				spec := d.Specs[0].(*ast.TypeSpec)

				if spec.Name.Name == *clientName {
					client := newBindingGenData(*clientName)
					if structDecl, ok := spec.Type.(*ast.InterfaceType); ok {
						parseMethods(&client, structDecl, fset)
					}

				}
			default:
			}
		}
		return true
	})
}

func parseMethods(client *BindingGenData, interDecl *ast.InterfaceType, fset *token.FileSet) {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	for _, field := range interDecl.Methods.List {
		identType, ok := field.Type.(*ast.Ident)
		if ok {
			if decl, ok := identType.Obj.Decl.(*ast.TypeSpec); ok {
				if inte, ok := decl.Type.(*ast.InterfaceType); ok {
					parseMethods(client, inte, fset)
					continue
				}
			}
			panic(*clientName + " should have only functional methods  or embedded interfaces")
		}
		funcType, ok := field.Type.(*ast.FuncType)
		if !ok {
			panic(*clientName + " should have only functional methods  or embedded interfaces")
		}
		requestArguments := []Field{}
		for _, param := range funcType.Params.List {
			a := Field{
				Name: getName(param),
				Type: getTypeName(fset, param.Type),
			}
			requestArguments = append(requestArguments, a)
		}
		results := []Field{}
		for _, param := range funcType.Results.List {
			results = append(results, Field{
				Name: getName(param),
				Type: getTypeName(fset, param.Type),
			})
		}

		for i, comment := range field.Doc.List {
			address, isEvent, isAuthTokenRequired, tokenScope := findBindAddress(comment)
			if isAuthTokenRequired {
				client.IsAuthTokenRequired = true
			}
			if tokenScope != "" {
				client.IsTokenScopePresent = true
			}

			if i < 1 {
				client.Methods = append(client.Methods, Method{
					Name:                field.Names[0].Name,
					Request:             requestArguments[1],
					Response:            results[0],
					Address:             address,
					IsEvent:             isEvent,
					IsAuthTokenRequired: isAuthTokenRequired,
					TokenScope:          tokenScope,
				})
			}

			client.MethodsForRegisterServer = append(client.MethodsForRegisterServer, Method{
				Name:                field.Names[0].Name,
				Request:             requestArguments[1],
				Response:            results[0],
				Address:             address,
				IsEvent:             isEvent,
				IsAuthTokenRequired: isAuthTokenRequired,
				TokenScope:          tokenScope,
			})
		}
	}

	template := template.Must(template.New("client").Parse(templateCode))

	f, err := os.Create(cwd + "/bind-gen.go")
	if err != nil {
		panic(err)
	}

	defer f.Close()

	err = template.Execute(f, client)
	if err != nil {
		panic(err)
	}
}

func findBindAddress(comment *ast.Comment) (address string, isEvent bool, isAuthTokenRequired bool, tokenScope string) {
	const methodPrefix = "// bind-method: "
	const eventPrefix = "// bind-event: "
	const authTokenPrefix = "auth_token="

	parts := strings.Split(comment.Text, "?")
	basePart := parts[0]

	if strings.HasPrefix(basePart, methodPrefix) {
		address = strings.TrimPrefix(basePart, methodPrefix)
		isEvent = false
	} else if strings.HasPrefix(basePart, eventPrefix) {
		address = strings.TrimPrefix(basePart, eventPrefix)
		isEvent = true
	}

	if len(parts) > 1 {
		paramsPart := parts[1]
		if strings.HasPrefix(paramsPart, authTokenPrefix) {
			isAuthTokenRequired = true
			tokenScope = strings.TrimPrefix(paramsPart, authTokenPrefix)
			if tokenScope == "required" {
				tokenScope = ""
			}
		}
	}

	return address, isEvent, isAuthTokenRequired, tokenScope
}

func getName(p *ast.Field) (n string) {
	if len(p.Names) == 0 {
		return ""
	}
	return p.Names[0].Name
}

func getTypeName(fset *token.FileSet, exp ast.Expr) (n string) {
	var tmp bytes.Buffer
	printer.Fprint(&tmp, fset, exp)
	return string(tmp.Bytes())
}

var templateCode = `
//nolint:unparam
// Code generated by genbind tool. DO NOT EDIT.
// source: {{.Source}}

package {{.Package}}

import (
	"context"
	
	"github.com/geruz/rizotto/bb"
)

{{if .IsAuthTokenRequired}}{{if .IsTokenScopePresent}}
func RegisterServer(srv {{.ServiceName}}, tokens map[string]auth.AuthToken) {
	{{range .MethodsForRegisterServer}}{{if .IsAuthTokenRequired}}bb.MustRegister("{{.Address}}", auth.RequiredTokenAuth(srv.{{.Name}}, tokens["{{.TokenScope}}"])){{else}}{{if .IsEvent}}bb.MustRegisterEvent("{{.Address}}", srv.{{.Name}}){{else}}bb.MustRegister("{{.Address}}", srv.{{.Name}}){{end}}{{end}}
	{{end}}
{{else}}
func RegisterServer(srv {{.ServiceName}}, authToken string) {
	{{range .MethodsForRegisterServer}}{{if .IsAuthTokenRequired}}bb.MustRegister("{{.Address}}", auth.RequiredBoaAuth(srv.{{.Name}}, authToken)){{else}}{{if .IsEvent}}bb.MustRegisterEvent("{{.Address}}", srv.{{.Name}}){{else}}bb.MustRegister("{{.Address}}", srv.{{.Name}}){{end}}{{end}}
	{{end}}
{{end}}{{else}}
func RegisterServer(srv {{.ServiceName}}) {
	{{range .MethodsForRegisterServer}}{{if .IsEvent}}bb.MustRegisterEvent("{{.Address}}", srv.{{.Name}}){{else}}bb.MustRegister("{{.Address}}", srv.{{.Name}}){{end}}
	{{end}}
{{end}}}

type (
	{{range .Methods}}{{.Name}} func(ctx context.Context, req {{.Request.Type}}) ({{.Response.Type}}, bb.ServiceError)
	{{end}}
)

type {{.ClientName}} interface {
	{{range .Methods}}{{.Name}}(ctx context.Context, req {{.Request.Type}}) ({{.Response.Type}}, bb.ServiceError)
	{{end}}
}


type {{.ClientImplementationName}} struct {
{{range .Methods}}    _{{.Name}} {{.Name}}
{{end}}
}

{{range .Methods}}func (c {{$.ClientImplementationName}}) {{.Name}}(ctx context.Context, req {{.Request.Type}}) ({{.Response.Type}}, bb.ServiceError) {
	return c._{{.Name}}(ctx, req)
}
{{end}}

func BindClient() {{.ClientName}} {
	return {{.ClientImplementationName}}{
		{{range .MethodsForRegisterServer}}{{if not .IsEvent}}_{{.Name}}: bb.MustBind[{{.Name}}]("{{.Address}}"),
		{{else}}//_{{.Name}}: bb.MustRegisterEvent[{{.Name}}]("{{.Address}}),
		{{end}}{{end}}
	}
}
`
