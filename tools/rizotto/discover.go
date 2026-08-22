package main

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// serviceRPC is one method found in the client interface of a service.
type serviceRPC struct {
	// Key is the CRUD method it implements: create, read, list, update or delete.
	Key string
	// Name is the method name, e.g. GetOrderRPC.
	Name string
	// Request is the request type of the method, e.g. GetOrderRequest.
	Request string
	// Response is the answer type of the method, e.g. Order.
	Response string
}

// serviceInfo describes a service already present in the project.
type serviceInfo struct {
	// Lower is the directory name of the service, e.g. order.
	Lower string
	// Package is the package name of its client, e.g. order.
	Package string
	// Import is the import path of the client package.
	Import string
	// Entity is the name of the entity the service serves, e.g. Order.
	Entity string
	// RPCs are the CRUD methods of the client interface, keyed by CRUD method.
	RPCs map[string]serviceRPC
}

var (
	errNoServices    = errors.New("no service found, add one with: rizotto service add")
	errUnknownSvc    = errors.New("unknown service")
	errNoClientIface = errors.New("no <Name>Client interface found in the client package")
)

// rpcPrefixes maps the method name prefix used by the scaffold onto a CRUD method.
//
//nolint:gochecknoglobals // the prefix table is static
var rpcPrefixes = map[string]string{
	"Create": methodCreate,
	"Get":    methodRead,
	"Select": methodList,
	"Update": methodUpdate,
	"Delete": methodDelete,
}

// discoverServices lists the services of the project, in directory order.
func discoverServices(projectRoot string, module string) ([]serviceInfo, error) {
	entries, err := os.ReadDir(filepath.Join(projectRoot, "services"))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("read services directory: %w", err)
	}

	services := []serviceInfo{}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		info, err := readService(projectRoot, module, entry.Name())
		if err != nil || info == nil {
			continue // a directory that does not look like a service is simply skipped
		}

		services = append(services, *info)
	}

	return services, nil
}

// findService returns the service with the given directory name.
func findService(services []serviceInfo, name string) (serviceInfo, error) {
	lower := strings.ToLower(strings.TrimSpace(name))

	for _, service := range services {
		if service.Lower == lower {
			return service, nil
		}
	}

	return serviceInfo{}, fmt.Errorf("%w %q, known services: %s", errUnknownSvc, name, serviceNames(services))
}

func serviceNames(services []serviceInfo) string {
	names := make([]string, 0, len(services))
	for _, service := range services {
		names = append(names, service.Lower)
	}

	return strings.Join(names, ", ")
}

// readService parses the client package of one service directory.
func readService(projectRoot string, module string, dir string) (*serviceInfo, error) {
	clientDir := filepath.Join(projectRoot, "services", dir, dir+"-client")

	file, err := parser.ParseFile(
		token.NewFileSet(), filepath.Join(clientDir, "client.go"), nil, parser.SkipObjectResolution,
	)
	if err != nil {
		return nil, err
	}

	entity, rpcs, err := clientInterface(file)
	if err != nil {
		return nil, err
	}

	return &serviceInfo{
		Lower:   dir,
		Package: file.Name.Name,
		Import:  module + "/services/" + dir + "/" + dir + "-client",
		Entity:  entity,
		RPCs:    rpcs,
	}, nil
}

// clientInterface finds the "<Entity>Client" interface of the file and reads its
// CRUD methods.
func clientInterface(file *ast.File) (string, map[string]serviceRPC, error) {
	for _, decl := range file.Decls {
		spec, iface, ok := interfaceDecl(decl)
		if !ok || !strings.HasSuffix(spec.Name.Name, "Client") {
			continue
		}

		return strings.TrimSuffix(spec.Name.Name, "Client"), interfaceRPCs(iface), nil
	}

	return "", nil, errNoClientIface
}

func interfaceDecl(decl ast.Decl) (*ast.TypeSpec, *ast.InterfaceType, bool) {
	generic, ok := decl.(*ast.GenDecl)
	if !ok || generic.Tok != token.TYPE {
		return nil, nil, false
	}

	for _, spec := range generic.Specs {
		typeSpec, ok := spec.(*ast.TypeSpec)
		if !ok {
			continue
		}

		iface, ok := typeSpec.Type.(*ast.InterfaceType)
		if ok {
			return typeSpec, iface, true
		}
	}

	return nil, nil, false
}

func interfaceRPCs(iface *ast.InterfaceType) map[string]serviceRPC {
	rpcs := map[string]serviceRPC{}

	for _, method := range iface.Methods.List {
		funcType, ok := method.Type.(*ast.FuncType)
		if !ok || len(method.Names) == 0 {
			continue
		}

		name := method.Names[0].Name

		key, ok := rpcKey(name)
		if !ok {
			continue
		}

		request, response, ok := requestAndResponse(funcType)
		if !ok {
			continue
		}

		rpcs[key] = serviceRPC{Key: key, Name: name, Request: request, Response: response}
	}

	return rpcs
}

// rpcKey maps "GetOrderRPC" onto "read".
func rpcKey(name string) (string, bool) {
	if !strings.HasSuffix(name, "RPC") {
		return "", false
	}

	for prefix, key := range rpcPrefixes {
		if strings.HasPrefix(name, prefix) {
			return key, true
		}
	}

	return "", false
}

// requestAndResponse reads the request and answer types of an RPC declared as
// func(ctx context.Context, req Request) (Response, bb.ServiceError).
func requestAndResponse(funcType *ast.FuncType) (string, string, bool) {
	const arguments = 2

	if funcType.Params == nil || len(funcType.Params.List) != arguments {
		return "", "", false
	}

	if funcType.Results == nil || len(funcType.Results.List) != arguments {
		return "", "", false
	}

	return typeName(funcType.Params.List[1].Type), typeName(funcType.Results.List[0].Type), true
}

func typeName(expr ast.Expr) string {
	var buf bytes.Buffer

	_ = printer.Fprint(&buf, token.NewFileSet(), expr)

	return buf.String()
}
