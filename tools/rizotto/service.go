package main

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
)

// serviceMethod is one RPC of a service, as seen by genbind.
type serviceMethod struct {
	Name     string
	Request  string
	Response string
	Address  string
}

// serviceSpec is the data the service templates are rendered with.
type serviceSpec struct {
	// Module is the go module path of the project the service belongs to.
	Module string
	// RizottoModule is the import path of the framework.
	RizottoModule string
	// Name is the exported name of the service entity, e.g. Order.
	Name string
	// NamePlural is the exported plural, e.g. Orders.
	NamePlural string
	// Lower is the package and directory name, e.g. order.
	Lower string
	// LowerPlural is the table name and the plural field name, e.g. orders.
	LowerPlural string
	// Package is the package of the implementation, e.g. ordersrv.
	Package string
	// DB tells whether the service owns a database table.
	DB bool

	Create bool
	Read   bool
	List   bool
	Update bool
	Delete bool

	// NeedsID is true when a method takes an entity id.
	NeedsID bool
	// NeedsModel is true when a method converts a database row into the client type.
	NeedsModel bool

	// Methods drives the client interface and the generated bindings.
	Methods []serviceMethod
}

const (
	methodCreate = "create"
	methodRead   = "read"
	methodList   = "list"
	methodUpdate = "update"
	methodDelete = "delete"
	// allMethods is the answer selecting every method.
	allMethods = "all"
)

// crudMethods are the answers accepted by the "which CRUD methods" question, in
// the order they are generated.
//
//nolint:gochecknoglobals // the method list is static
var crudMethods = []struct {
	key     string
	aliases []string
}{
	{key: methodCreate, aliases: []string{"c", methodCreate, "add", "insert"}},
	{key: methodRead, aliases: []string{"r", methodRead, "get"}},
	{key: methodList, aliases: []string{"l", methodList, "select"}},
	{key: methodUpdate, aliases: []string{"u", methodUpdate, "edit"}},
	{key: methodDelete, aliases: []string{"d", methodDelete, "del", "remove"}},
}

var (
	errNoMethods = errors.New("choose at least one method: create, read, list, update, delete (or all)")
	errNoService = errors.New("no go.mod found, run the command inside a project or pass -project")
)

// normalizeMethods turns "all", "crud" or "create, r,list" into the canonical
// "create,read,list,update,delete" order.
func normalizeMethods(raw string) (string, error) {
	answer := strings.ToLower(strings.TrimSpace(raw))
	if answer == allMethods || answer == "crud" || answer == "*" {
		return methodKeys(), nil
	}

	chosen, err := chosenMethods(answer)
	if err != nil {
		return "", err
	}

	if len(chosen) == 0 {
		return "", errNoMethods
	}

	selected := make([]string, 0, len(crudMethods))

	for _, method := range crudMethods {
		if chosen[method.key] {
			selected = append(selected, method.key)
		}
	}

	return strings.Join(selected, ","), nil
}

// chosenMethods maps every field of the answer onto a canonical method key.
func chosenMethods(answer string) (map[string]bool, error) {
	fields := strings.FieldsFunc(answer, func(r rune) bool {
		return r == ',' || r == ' ' || r == '+' || r == ';'
	})

	chosen := make(map[string]bool, len(fields))

	for _, field := range fields {
		key, ok := methodKey(field)
		if !ok {
			return nil, fmt.Errorf("%w, got %q", errNoMethods, field)
		}

		chosen[key] = true
	}

	return chosen, nil
}

func methodKey(field string) (string, bool) {
	for _, method := range crudMethods {
		if slices.Contains(method.aliases, field) {
			return method.key, true
		}
	}

	return "", false
}

func methodKeys() string {
	keys := make([]string, 0, len(crudMethods))
	for _, method := range crudMethods {
		keys = append(keys, method.key)
	}

	return strings.Join(keys, ",")
}

func newServiceSpec(module, name, methods string, withDB bool) (serviceSpec, error) {
	name, err := normalizeName(name)
	if err != nil {
		return serviceSpec{}, err
	}

	methods, err = normalizeMethods(methods)
	if err != nil {
		return serviceSpec{}, err
	}

	name = strings.ToUpper(name[:1]) + name[1:]
	lower := strings.ToLower(name)
	chosen := strings.Split(methods, ",")

	spec := serviceSpec{
		Module:        module,
		RizottoModule: rizottoModule,
		Name:          name,
		NamePlural:    pluralize(name),
		Lower:         lower,
		LowerPlural:   pluralize(lower),
		Package:       lower + "srv",
		DB:            withDB,
		Create:        slices.Contains(chosen, methodCreate),
		Read:          slices.Contains(chosen, methodRead),
		List:          slices.Contains(chosen, methodList),
		Update:        slices.Contains(chosen, methodUpdate),
		Delete:        slices.Contains(chosen, methodDelete),
		NeedsID:       false,
		NeedsModel:    false,
		Methods:       nil,
	}
	spec.NeedsID = spec.Read || spec.Update || spec.Delete
	spec.NeedsModel = spec.Create || spec.Read || spec.List || spec.Update
	spec.Methods = spec.buildMethods()

	return spec, nil
}

// Dir is the directory of the service, relative to the project root.
func (s serviceSpec) Dir() string {
	return "services/" + s.Lower
}

// Constructor is the call registering the service in server.go.
func (s serviceSpec) Constructor() string {
	if s.DB {
		return s.Package + ".New" + s.Name + "Service(pgPool)"
	}

	return s.Package + ".New" + s.Name + "Service()"
}

func (s serviceSpec) buildMethods() []serviceMethod {
	address := func(action string) string {
		return "http://" + s.Lower + "-service/" + s.Lower + "/" + action
	}

	methods := []serviceMethod{}

	if s.Create {
		methods = append(methods, serviceMethod{
			Name:     "Create" + s.Name + "RPC",
			Request:  "Create" + s.Name + "Request",
			Response: s.Name,
			Address:  address("create"),
		})
	}

	if s.Read {
		methods = append(methods, serviceMethod{
			Name:     "Get" + s.Name + "RPC",
			Request:  "Get" + s.Name + "Request",
			Response: s.Name,
			Address:  address("get"),
		})
	}

	if s.List {
		methods = append(methods, serviceMethod{
			Name:     "Select" + s.NamePlural + "RPC",
			Request:  "Select" + s.NamePlural + "Request",
			Response: s.Name + "List",
			Address:  address("select"),
		})
	}

	if s.Update {
		methods = append(methods, serviceMethod{
			Name:     "Update" + s.Name + "RPC",
			Request:  "Update" + s.Name + "Request",
			Response: s.Name,
			Address:  address("update"),
		})
	}

	if s.Delete {
		methods = append(methods, serviceMethod{
			Name:     "Delete" + s.Name + "RPC",
			Request:  "Delete" + s.Name + "Request",
			Response: "Deleted" + s.Name,
			Address:  address("delete"),
		})
	}

	return methods
}

// files lists what the service is made of.
func (s serviceSpec) files() []fileSpec {
	const withRepository = 10

	dir := s.Dir()
	files := make([]fileSpec, 0, withRepository)
	files = append(files,
		fileSpec{tmpl: "service/service.go.tmpl", out: dir + "/" + s.Lower + "-service.go", sqlOnly: always},
		fileSpec{tmpl: "service/client.go.tmpl", out: dir + "/" + s.Lower + "-client/client.go", sqlOnly: always},
		fileSpec{tmpl: "service/bind-gen.go.tmpl", out: dir + "/" + s.Lower + "-client/bind-gen.go", sqlOnly: always},
	)

	if !s.DB {
		return files
	}

	return append(files,
		fileSpec{
			tmpl:    "service/repository.go.tmpl",
			out:     dir + "/repository/" + s.Lower + "-repository.go",
			sqlOnly: always,
		},
		fileSpec{tmpl: "service/schema.sql.tmpl", out: dir + "/repository/sql/schema.sql", sqlOnly: always},
		fileSpec{
			tmpl:    "service/queries.sql.tmpl",
			out:     dir + "/repository/sql/" + s.Lower + "-queries.sql",
			sqlOnly: always,
		},
		fileSpec{
			tmpl:    "service/migration.sql.tmpl",
			out:     dir + "/repository/migrations/000001_create_" + s.LowerPlural + ".sql",
			sqlOnly: always,
		},
		fileSpec{tmpl: "service/db/db.go.tmpl", out: dir + "/repository/db/db.go", sqlOnly: always},
		fileSpec{tmpl: "service/db/models.go.tmpl", out: dir + "/repository/db/models.go", sqlOnly: always},
		fileSpec{
			tmpl:    "service/db/queries.sql.go.tmpl",
			out:     dir + "/repository/db/" + s.Lower + "-queries.sql.go",
			sqlOnly: always,
		},
	)
}

// generateService writes the service into an existing project and registers its
// queries in sqlc.yaml.
func generateService(spec serviceSpec, projectRoot string) ([]string, error) {
	written, err := generate(spec, spec.files(), projectRoot)
	if err != nil {
		return nil, err
	}

	if !spec.DB {
		return written, nil
	}

	changed, err := addSqlcEntry(projectRoot, spec)
	if err != nil {
		return nil, err
	}

	if changed {
		written = append(written, sqlcConfigFile)
	}

	sort.Strings(written)

	return written, nil
}

// pluralize is good enough for the entity names the scaffold deals with.
func pluralize(word string) string {
	lower := strings.ToLower(word)

	switch {
	case strings.HasSuffix(lower, "s"), strings.HasSuffix(lower, "x"), strings.HasSuffix(lower, "z"),
		strings.HasSuffix(lower, "ch"), strings.HasSuffix(lower, "sh"):
		return word + "es"
	case strings.HasSuffix(lower, "y") && len(word) > 1 && !isVowel(lower[len(lower)-2]):
		return word[:len(word)-1] + "ies"
	default:
		return word + "s"
	}
}

func isVowel(letter byte) bool {
	return strings.IndexByte("aeiou", letter) >= 0
}
