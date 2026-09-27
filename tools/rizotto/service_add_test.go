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
	projectFlag  = "-project"
	dbFlag       = "-db"
	methodsFlag  = "-methods"
	testService  = "Order"
	otherService = "Category"
	// serviceLower is the directory testService lands in.
	serviceLower = "order"
	// unknownSubcommand is not a subcommand of service or controller.
	unknownSubcommand = "remove"

	caseInvalidName       = "invalid name"
	casePositional        = "positional arguments"
	caseMissingSubcommand = "missing subcommand"

	// pluralName and singularName are the pair the plural check is exercised
	// with; boxName and addressName are the awkward ones pluralize has rules for.
	pluralName   = "Articles"
	singularName = "Article"
	boxName      = "Box"
	addressName  = "Address"
)

func Test_runService_AsksEveryStep(t *testing.T) {
	t.Parallel()

	root := newProjectDir(t)
	out := &bytes.Buffer{}
	answers := strings.Join([]string{testService, "y", allMethods, ""}, "\n")

	err := runService([]string{addCommand, projectFlag, root}, strings.NewReader(answers), out)
	must.NoError(t, err)

	printed := out.String()
	must.StrContains(t, printed, "Service name")
	must.StrContains(t, printed, "Does the service work with the database")
	must.StrContains(t, printed, "CRUD methods (create, read, list, update, delete) [all]")
	must.StrContains(t, printed, `Added service "Order" (with the orders table)`)
	must.StrContains(t, printed, "Methods: CreateOrderRPC, GetOrderRPC, SelectOrdersRPC, UpdateOrderRPC, DeleteOrderRPC")
	must.StrContains(t, printed, "order.RegisterServer(ordersrv.NewOrderService(pgPool))")
	must.StrContains(t, printed, "task migrate-up")

	must.FileExists(t, filepath.Join(root, "services", "order", "order-service.go"))
	must.FileExists(t, filepath.Join(root, sqlcConfigFile))
}

func Test_runService_FlagsSkipTheQuestions(t *testing.T) {
	t.Parallel()

	root := newProjectDir(t)
	out := &bytes.Buffer{}
	args := []string{addCommand, projectFlag, root, nameFlag, otherService, dbFlag, "no", methodsFlag, readAndList}

	err := runService(args, strings.NewReader(""), out)
	must.NoError(t, err)

	printed := out.String()
	must.StrNotContains(t, printed, "Service name")
	must.StrContains(t, printed, `Added service "Category" (in memory)`)
	must.StrContains(t, printed, "Methods: GetCategoryRPC, SelectCategoriesRPC")
	must.StrContains(t, printed, "category.RegisterServer(categorysrv.NewCategoryService())")

	_, err = os.Stat(filepath.Join(root, sqlcConfigFile))
	must.True(t, os.IsNotExist(err))

	// the service directory is not overwritten silently
	err = runService(args, strings.NewReader(""), out)
	must.ErrorIs(t, err, errServiceExist)
}

func Test_runService_FindsTheProjectFromASubdirectory(t *testing.T) {
	t.Parallel()

	root := newProjectDir(t)
	nested := filepath.Join(root, "api", "doc")

	err := os.MkdirAll(nested, dirPerm)
	must.NoError(t, err)

	out := &bytes.Buffer{}
	args := []string{addCommand, projectFlag, nested, nameFlag, testService, dbFlag, "no", methodsFlag, "create"}

	err = runService(args, strings.NewReader(""), out)
	must.NoError(t, err)
	must.FileExists(t, filepath.Join(root, "services", "order", "order-service.go"))
	must.StrContains(t, readFile(t, root, "services/order/order-client/client.go"), "package order")
}

func Test_findProject_FindsTheServerModuleFromTheRepository(t *testing.T) {
	t.Parallel()

	repo := t.TempDir()
	server := filepath.Join(repo, serverDir)
	web := filepath.Join(repo, webDir, "src")

	for _, dir := range []string{server, web} {
		must.NoError(t, os.MkdirAll(dir, dirPerm))
	}

	err := os.WriteFile(filepath.Join(server, "go.mod"), []byte("module "+testModule+"\n"), filePerm)
	must.NoError(t, err)

	for _, start := range []string{repo, server, web} {
		root, module, err := findProject(start)
		must.NoError(t, err, must.Sprintf("start %s", start))
		must.Eq(t, server, root, must.Sprintf("start %s", start))
		must.Eq(t, testModule, module)
	}
}

func Test_runService_RejectsBadInput(t *testing.T) {
	t.Parallel()

	root := newProjectDir(t)

	cases := map[string][]string{
		caseMissingSubcommand: {},
		"unknown subcommand":  {unknownSubcommand, nameFlag, testService},
		caseInvalidName:       {addCommand, projectFlag, root, nameFlag, invalidName, dbFlag, "no", methodsFlag, allMethods},
		"invalid db answer":   {addCommand, projectFlag, root, nameFlag, testService, dbFlag, "maybe"},
		"invalid methods":     {addCommand, projectFlag, root, nameFlag, testService, dbFlag, "no", methodsFlag, "drop"},
		casePositional:        {addCommand, testService},
	}

	for name, args := range cases {
		out := &bytes.Buffer{}

		err := runService(args, strings.NewReader(""), out)
		must.ErrorIs(t, err, errUsage, must.Sprintf("case %q", name))
	}
}

func Test_runService_RejectsAPluralName(t *testing.T) {
	t.Parallel()

	root := newProjectDir(t)
	out := &bytes.Buffer{}
	args := []string{addCommand, projectFlag, root, nameFlag, pluralName, dbFlag, "yes", methodsFlag, allMethods}

	err := runService(args, strings.NewReader(""), out)
	must.ErrorIs(t, err, errPluralName)
	must.StrContains(t, err.Error(), `"`+singularName+`"`)

	// Nothing is written, so a retry with the singular is not blocked by the
	// "directory already exists" guard.
	must.DirNotExists(t, filepath.Join(root, "services", "articles"))
	must.DirNotExists(t, filepath.Join(root, "services", "article"))
}

// Test_runService_ReAsksAfterAPluralName covers the interactive path: askValid
// prints the suggestion and asks again rather than giving up.
func Test_runService_ReAsksAfterAPluralName(t *testing.T) {
	t.Parallel()

	root := newProjectDir(t)
	out := &bytes.Buffer{}
	answers := strings.Join([]string{pluralName, singularName, "y", allMethods, ""}, "\n")

	err := runService([]string{addCommand, projectFlag, root}, strings.NewReader(answers), out)
	must.NoError(t, err)

	printed := out.String()
	must.StrContains(t, printed, `"`+pluralName+`" looks plural, try "`+singularName+`"`)
	must.StrContains(t, printed, `Added service "`+singularName+`" (with the articles table)`)
	must.FileExists(t, filepath.Join(root, "services", "article", "article-service.go"))
}

func Test_runService_NeedsAProject(t *testing.T) {
	t.Parallel()

	out := &bytes.Buffer{}
	args := []string{addCommand, projectFlag, t.TempDir(), nameFlag, testService, dbFlag, "no", methodsFlag, allMethods}

	err := runService(args, strings.NewReader(""), out)
	must.ErrorIs(t, err, errNoService)
}

// newProjectDir creates the smallest directory findProject accepts.
func newProjectDir(t *testing.T) string {
	t.Helper()

	root := t.TempDir()

	err := os.WriteFile(
		filepath.Join(root, "go.mod"),
		[]byte("module "+testModule+"\n\ngo 1.25.0\n"),
		filePerm,
	)
	must.NoError(t, err)

	return root
}
