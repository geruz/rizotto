package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shoenig/test/must"
)

const (
	serviceFlag  = "-service"
	authFlag     = "-auth"
	prefixFlag   = "-prefix"
	ordersPrefix = "/api/v1/orders"
)

func Test_runController_AsksEveryStep(t *testing.T) {
	t.Parallel()

	root := newProjectWithService(t, allMethods)
	out := &bytes.Buffer{}
	answers := strings.Join([]string{"", "", readAndList, "n", "/public/orders", ""}, "\n")

	err := runController([]string{addCommand, projectFlag, root, skipTidyFlag}, strings.NewReader(answers), out)
	must.NoError(t, err)

	printed := out.String()
	must.StrContains(t, printed, "Service to expose ("+serviceLower+") ["+serviceLower+"]")
	must.StrContains(t, printed, "Controller name [Order]")
	must.StrContains(t, printed, "Routes (create, read, list, update, delete) [create,read,list,update,delete]")
	must.StrContains(t, printed, "Do the routes need an authenticated user (PrivateArea)? [Y/n]")
	must.StrContains(t, printed, "Route prefix ["+ordersPrefix+"]")
	must.StrContains(t, printed, `Added controller "Order" (PublicArea)`)
	must.StrContains(t, printed, "GET /public/orders/{order_id}")
	must.StrContains(t, printed, "orderapi.NewOrderController().RouteTable()")

	controller := readFile(t, root, "api/order-api/order.go")
	must.StrContains(t, controller, "api.PublicArea")
	must.StrNotContains(t, controller, "createOrder")
}

func Test_runController_FlagsSkipTheQuestions(t *testing.T) {
	t.Parallel()

	root := newProjectWithService(t, allMethods)
	out := &bytes.Buffer{}
	args := []string{
		addCommand, projectFlag, root, serviceFlag, serviceLower,
		nameFlag, testService, methodsFlag, allMethods, authFlag, "yes", prefixFlag, "/api/v2/orders", skipTidyFlag,
	}

	err := runController(args, strings.NewReader(""), out)
	must.NoError(t, err)
	must.StrNotContains(t, out.String(), "Service to expose")
	must.FileExists(t, filepath.Join(root, "api", "order-api", "order.go"))
	must.FileExists(t, filepath.Join(root, "api", "order-api", "order_test.go"))
	must.StrContains(t, readFile(t, root, "api/order-api/order.go"), `"POST /api/v2/orders"`)

	// the controller directory is not overwritten silently
	err = runController(args, strings.NewReader(""), out)
	must.ErrorIs(t, err, errControllerExist)
}

func Test_runController_RejectsBadInput(t *testing.T) {
	t.Parallel()

	root := newProjectWithService(t, readAndList)
	valid := []string{addCommand, projectFlag, root, serviceFlag, serviceLower}

	cases := map[string][]string{
		"missing subcommand":  {},
		"unknown subcommand":  {unknownSubcommand},
		"unknown service":     {addCommand, projectFlag, root, serviceFlag, "invoice"},
		caseInvalidName:       append(append([]string{}, valid...), nameFlag, invalidName),
		"invalid auth answer": append(append([]string{}, valid...), authFlag, "maybe"),
		"invalid prefix":      append(append([]string{}, valid...), prefixFlag, "api v1"),
		casePositional:        {addCommand, testService},
	}

	for name, args := range cases {
		out := &bytes.Buffer{}

		err := runController(append(args, skipTidyFlag), strings.NewReader(""), out)
		must.ErrorIs(t, err, errUsage, must.Sprintf("case %q", name))
	}
}

func Test_runController_RejectsAPluralName(t *testing.T) {
	t.Parallel()

	root := newProjectWithService(t, allMethods)
	out := &bytes.Buffer{}
	args := []string{
		addCommand, projectFlag, root, serviceFlag, serviceLower,
		nameFlag, pluralName, methodsFlag, allMethods, skipTidyFlag,
	}

	err := runController(args, strings.NewReader(""), out)
	must.ErrorIs(t, err, errPluralName)
	must.StrContains(t, err.Error(), `"`+singularName+`"`)
}

func Test_runController_NeedsAService(t *testing.T) {
	t.Parallel()

	out := &bytes.Buffer{}
	args := []string{addCommand, projectFlag, newProjectDir(t), nameFlag, testService, skipTidyFlag}

	err := runController(args, strings.NewReader(""), out)
	must.ErrorIs(t, err, errNoServices)
}

func Test_runController_RejectsRoutesTheServiceCannotServe(t *testing.T) {
	t.Parallel()

	root := newProjectWithService(t, readAndList)
	out := &bytes.Buffer{}
	args := []string{addCommand, projectFlag, root, serviceFlag, serviceLower, methodsFlag, allMethods, skipTidyFlag}

	err := runController(args, strings.NewReader(""), out)
	must.ErrorIs(t, err, errNoRPC)
}

// newProjectWithService creates a project directory holding one generated service.
func newProjectWithService(t *testing.T, methods string) string {
	t.Helper()

	root := newProjectDir(t)

	spec, err := newServiceSpec(testModule, testService, methods, false)
	must.NoError(t, err)

	_, err = generateService(spec, root)
	must.NoError(t, err)

	return root
}
