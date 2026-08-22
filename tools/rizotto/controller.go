package main

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// controllerRoute is one HTTP route of a controller.
type controllerRoute struct {
	// Key is the CRUD method it exposes.
	Key string
	// Route is the route string, e.g. "GET /api/v1/orders/{order_id}".
	Route string
	// Handler is the controller method serving it, e.g. getOrder.
	Handler string
	// Field is the RPC field of the controller, e.g. getOrderRPC.
	Field string
	// RPC is the method of the service client the field is bound to.
	RPC serviceRPC
	// Description documents the route in the OpenAPI document.
	Description string
}

// controllerSpec is the data the controller templates are rendered with.
type controllerSpec struct {
	Module        string
	RizottoModule string
	// Name is the exported name of the controller entity, e.g. Order.
	Name string
	// NamePlural is the exported plural, e.g. Orders.
	NamePlural string
	// Lower is the lowercased name, e.g. order.
	Lower string
	// LowerPlural is the lowercased plural, e.g. orders.
	LowerPlural string
	// Package is the package of the controller, e.g. orderapi.
	Package string
	// Prefix is the route prefix, e.g. /api/v1/orders.
	Prefix string
	// Area is the context builder of api/controller.go: PrivateArea or PublicArea.
	Area string
	// Context is the context type the handlers take.
	Context string
	// Service is the service the controller exposes.
	Service serviceInfo

	Create bool
	Read   bool
	List   bool
	Update bool
	Delete bool

	// Routes drives the constructor, the tests and the documentation.
	Routes []controllerRoute
}

const (
	privateArea = "PrivateArea"
	publicArea  = "PublicArea"
)

var (
	errNoRPC      = errors.New("the service has no method for")
	errPrefixForm = errors.New(`the route prefix must look like /api/v1/orders`)

	prefixRe = regexp.MustCompile(`^(/[a-zA-Z0-9._~-]+)+$`)
)

// normalizePrefix validates the answer of the route prefix question.
func normalizePrefix(raw string) (string, error) {
	prefix := strings.TrimSpace(raw)
	if prefix != "/" {
		prefix = strings.TrimSuffix(prefix, "/")
	}

	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}

	if !prefixRe.MatchString(prefix) {
		return "", fmt.Errorf("%w, got %q", errPrefixForm, raw)
	}

	return prefix, nil
}

func newControllerSpec(
	module string,
	name string,
	service serviceInfo,
	methods string,
	prefix string,
	private bool,
) (controllerSpec, error) {
	name, err := normalizeEntityName(name)
	if err != nil {
		return controllerSpec{}, err
	}

	methods, err = normalizeMethods(methods)
	if err != nil {
		return controllerSpec{}, err
	}

	name = strings.ToUpper(name[:1]) + name[1:]
	lower := strings.ToLower(name)
	chosen := strings.Split(methods, ",")

	if prefix == "" {
		prefix = defaultPrefix(lower)
	}

	prefix, err = normalizePrefix(prefix)
	if err != nil {
		return controllerSpec{}, err
	}

	spec := controllerSpec{
		Module:        module,
		RizottoModule: rizottoModule,
		Name:          name,
		NamePlural:    pluralize(name),
		Lower:         lower,
		LowerPlural:   pluralize(lower),
		Package:       lower + "api",
		Prefix:        prefix,
		Area:          area(private),
		Context:       contextType(private),
		Service:       service,
		Create:        slices.Contains(chosen, methodCreate),
		Read:          slices.Contains(chosen, methodRead),
		List:          slices.Contains(chosen, methodList),
		Update:        slices.Contains(chosen, methodUpdate),
		Delete:        slices.Contains(chosen, methodDelete),
		Routes:        nil,
	}

	spec.Routes, err = spec.buildRoutes(chosen)
	if err != nil {
		return controllerSpec{}, err
	}

	return spec, nil
}

func defaultPrefix(lower string) string {
	return "/api/v1/" + pluralize(lower)
}

func area(private bool) string {
	if private {
		return privateArea
	}

	return publicArea
}

func contextType(private bool) string {
	if private {
		return "UserHTTPContext"
	}

	return "PublicHTTPContext"
}

// defaultMethods are the CRUD methods a service actually implements.
func defaultMethods(service serviceInfo) string {
	available := make([]string, 0, len(crudMethods))

	for _, method := range crudMethods {
		if _, ok := service.RPCs[method.key]; ok {
			available = append(available, method.key)
		}
	}

	return strings.Join(available, ",")
}

// Dir is the directory of the controller, relative to the project root.
func (c controllerSpec) Dir() string {
	return "api/" + c.Lower + "-api"
}

// RouteTable is the call joining the controller into server.go.
func (c controllerSpec) RouteTable() string {
	return c.Package + ".New" + c.Name + "Controller().RouteTable()"
}

// buildRoutes pairs every chosen CRUD method with a route and the RPC serving it.
func (c controllerSpec) buildRoutes(chosen []string) ([]controllerRoute, error) {
	ordered := make([]controllerRoute, 0, len(chosen))

	for _, method := range crudMethods {
		if !slices.Contains(chosen, method.key) {
			continue
		}

		rpc, ok := c.Service.RPCs[method.key]
		if !ok {
			return nil, fmt.Errorf("%w %s: %s", errNoRPC, method.key, c.Service.Lower)
		}

		ordered = append(ordered, c.routeFor(method.key, rpc))
	}

	return ordered, nil
}

// routeFor describes the HTTP route exposing one CRUD method.
func (c controllerSpec) routeFor(key string, rpc serviceRPC) controllerRoute {
	byID := c.Prefix + "/{" + c.Lower + "_id}"

	switch key {
	case methodCreate:
		return controllerRoute{
			Key: key, Route: "POST " + c.Prefix, Handler: "create" + c.Name,
			Field: "create" + c.Name + "RPC", RPC: rpc,
			Description: "Creates a `" + c.Name + "`.",
		}
	case methodList:
		return controllerRoute{
			Key: key, Route: "GET " + c.Prefix, Handler: "list" + c.NamePlural,
			Field: "select" + c.NamePlural + "RPC", RPC: rpc,
			Description: "Returns a page of `" + c.Name + "` objects.",
		}
	case methodUpdate:
		return controllerRoute{
			Key: key, Route: "PUT " + byID, Handler: "update" + c.Name,
			Field: "update" + c.Name + "RPC", RPC: rpc,
			Description: "Updates the `" + c.Name + "` with the given `id`.",
		}
	case methodDelete:
		return controllerRoute{
			Key: key, Route: "DELETE " + byID, Handler: "delete" + c.Name,
			Field: "delete" + c.Name + "RPC", RPC: rpc,
			Description: "Deletes the `" + c.Name + "` with the given `id`.",
		}
	default: // methodRead
		return controllerRoute{
			Key: key, Route: "GET " + byID, Handler: "get" + c.Name,
			Field: "get" + c.Name + "RPC", RPC: rpc,
			Description: "Returns the `" + c.Name + "` with the given `id`.",
		}
	}
}

// files lists what the controller is made of.
func (c controllerSpec) files() []fileSpec {
	dir := c.Dir()

	return []fileSpec{
		{tmpl: "controller/controller.go.tmpl", out: dir + "/" + c.Lower + ".go", sqlOnly: always},
		{tmpl: "controller/controller_test.go.tmpl", out: dir + "/" + c.Lower + "_test.go", sqlOnly: always},
	}
}

func generateController(spec controllerSpec, projectRoot string) ([]string, error) {
	return generate(spec, spec.files(), projectRoot)
}
