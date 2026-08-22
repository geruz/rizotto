package main

import (
	"testing"

	"github.com/shoenig/test/must"
)

func Test_normalizePrefix(t *testing.T) {
	t.Parallel()

	valid := map[string]string{
		ordersPrefix:        ordersPrefix,
		"  " + ordersPrefix: ordersPrefix,
		ordersPrefix + "/":  ordersPrefix,
		"api/v1/orders":     ordersPrefix,
		"/orders":           "/orders",
	}

	for raw, expected := range valid {
		normalized, err := normalizePrefix(raw)
		must.NoError(t, err, must.Sprintf("prefix %q", raw))
		must.Eq(t, expected, normalized, must.Sprintf("prefix %q", raw))
	}

	for _, raw := range []string{"", "/api/v1/{orders}", "/api v1", "//"} {
		_, err := normalizePrefix(raw)
		must.Error(t, err, must.Sprintf("prefix %q must be rejected", raw))
	}
}

func Test_newControllerSpec(t *testing.T) {
	t.Parallel()

	service := mustServiceInfo(t, allMethods)

	spec, err := newControllerSpec(testModule, testService, service, readAndList, "", true)
	must.NoError(t, err)
	must.Eq(t, "orderapi", spec.Package)
	must.Eq(t, "api/order-api", spec.Dir())
	must.Eq(t, ordersPrefix, spec.Prefix)
	must.Eq(t, privateArea, spec.Area)
	must.Eq(t, "UserHTTPContext", spec.Context)
	must.Eq(t, "orderapi.NewOrderController().RouteTable()", spec.RouteTable())

	must.Eq(t, []string{"GET " + ordersPrefix + "/{order_id}", "GET " + ordersPrefix}, routeStrings(spec))
	must.Eq(t, []string{"getOrder", "listOrders"}, handlers(spec))
	must.Eq(t, "GetOrderRPC", spec.Routes[0].RPC.Name)

	public, err := newControllerSpec(testModule, testService, service, allMethods, "/public/orders", false)
	must.NoError(t, err)
	must.Eq(t, publicArea, public.Area)
	must.Eq(t, "PublicHTTPContext", public.Context)
	must.Eq(t, []string{
		"POST /public/orders",
		"GET /public/orders/{order_id}",
		"GET /public/orders",
		"PUT /public/orders/{order_id}",
		"DELETE /public/orders/{order_id}",
	}, routeStrings(public))
}

func Test_newControllerSpec_RejectsRoutesTheServiceCannotServe(t *testing.T) {
	t.Parallel()

	service := mustServiceInfo(t, readAndList)

	_, err := newControllerSpec(testModule, testService, service, allMethods, "", true)
	must.ErrorIs(t, err, errNoRPC)

	must.Eq(t, readAndList, defaultMethods(service))
}

func Test_generateController(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	service := mustServiceInfo(t, allMethods)

	spec, err := newControllerSpec(testModule, testService, service, allMethods, "", true)
	must.NoError(t, err)

	files, err := generateController(spec, root)
	must.NoError(t, err)

	mustHaveFiles(t, root, files,
		"api/order-api/order.go",
		"api/order-api/order_test.go",
	)

	controller := readFile(t, root, "api/order-api/order.go")
	must.StrContains(t, controller, "package orderapi")
	must.StrContains(t, controller, `gateway.JSONMethod("POST `+ordersPrefix+`", api.PrivateArea, ctrl.createOrder)`)
	must.StrContains(t, controller,
		`gateway.JSONMethod("DELETE `+ordersPrefix+`/{order_id}", api.PrivateArea, ctrl.deleteOrder)`)
	must.StrContains(t, controller, "order.GetOrderRPC")
	must.StrContains(t, controller, `in:"path=order_id" json:"-"`)
	must.StrContains(t, controller, "IfNotFound(orderNotFound)")

	test := readFile(t, root, "api/order-api/order_test.go")
	must.StrContains(t, test, "func Test_CreateOrderRoute(t *testing.T)")
	must.StrContains(t, test, "cfg.WithJSONBody(CreateOrderRequest{Name: testOrderName})")
	must.StrContains(t, test, `api.GroupRouteTests(t, "GET `+ordersPrefix+`/{order_id}", NewOrderController)`)
}

func Test_generateController_OnlyTheChosenRoutes(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	service := mustServiceInfo(t, allMethods)

	spec, err := newControllerSpec(testModule, testService, service, methodList, "", false)
	must.NoError(t, err)

	_, err = generateController(spec, root)
	must.NoError(t, err)

	controller := readFile(t, root, "api/order-api/order.go")
	must.StrContains(t, controller, "listOrders")
	must.StrContains(t, controller, "api.PublicHTTPContext")
	must.StrNotContains(t, controller, "createOrder")
	must.StrNotContains(t, controller, "deleteOrder")
	must.StrNotContains(t, controller, "orderNotFound")
}

// mustServiceInfo builds the service description discovery would produce for a
// service generated with the given methods.
func mustServiceInfo(t *testing.T, methods string) serviceInfo {
	t.Helper()

	root := t.TempDir()

	spec, err := newServiceSpec(testModule, testService, methods, false)
	must.NoError(t, err)

	_, err = generateService(spec, root)
	must.NoError(t, err)

	services, err := discoverServices(root, testModule)
	must.NoError(t, err)
	must.Eq(t, 1, len(services))

	return services[0]
}

func routeStrings(spec controllerSpec) []string {
	routes := make([]string, 0, len(spec.Routes))
	for _, route := range spec.Routes {
		routes = append(routes, route.Route)
	}

	return routes
}

func handlers(spec controllerSpec) []string {
	names := make([]string, 0, len(spec.Routes))
	for _, route := range spec.Routes {
		names = append(names, route.Handler)
	}

	return names
}
