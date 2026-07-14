package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/geruz/rizotto/documentation/openapi"
	"github.com/geruz/rizotto/gateway"
	"github.com/shoenig/test/must"
)

type GroupRouteTestRunner[Controller controller] struct {
	t        *testing.T
	RouteStr string
	// HandlerBase
	Constructor func() Controller

	doc *openapi.Operation
}

type controller interface {
	RouteTable() gateway.RouteTable
}

func GroupRouteTests[Controller controller](
	t *testing.T,
	routeStr string,
	ctrlConstructor func() Controller,
) GroupRouteTestRunner[Controller] {
	t.Helper()

	return GroupRouteTestRunner[Controller]{
		t:           t,
		RouteStr:    routeStr,
		Constructor: ctrlConstructor,
		doc:         nil,
	}
}

type CallInfo[Request any, Response any] struct {
	request        Request
	actualResponse Response
	actualError    gateway.HTTPError
	//	result         Result[Response]
	response *http.Response
	body     []byte

	doc *openapi.Operation
}

func (c CallInfo[TRequest, TResponse]) AddInDocumentation(t *testing.T, desc string) CallInfo[TRequest, TResponse] {
	t.Helper()
	err := c.doc.AddRequest(desc, c.request)
	if err != nil {
		t.Fatal("Error while adding request to documentation. Original error: ", err.Error())
	}

	if c.actualError != nil {
		err := c.doc.AddResponse(desc, c.actualError.StatusCode(), c.actualError)
		if err != nil {
			t.Fatal("Error while adding response to documentation. Original error: ", err.Error())
		}

		return c
	}

	err = c.doc.AddResponse(desc, c.response.StatusCode, c.actualResponse)
	if err != nil {
		t.Fatal("Error while adding response to documentation. Original error: ", err.Error())
	}

	return c
}

func (c CallInfo[TRequest, TResponse]) ExpectedRequest(t *testing.T, data TRequest) CallInfo[TRequest, TResponse] {
	t.Helper()
	must.Eq(t, c.request, data)

	return c
}

func (c CallInfo[TRequest, TResponse]) ExpectedResponse(t *testing.T, data TResponse) CallInfo[TRequest, TResponse] {
	t.Helper()
	c.ExpectedSuccess(t, http.StatusOK, data)

	return c
}

func (c CallInfo[TRequest, TResponse]) ExpectedAnswer(t *testing.T, code int, data any) CallInfo[TRequest, TResponse] {
	t.Helper()
	c.ExpectedBody(t, code, data)

	return c
}

func (c CallInfo[TRequest, TResponse]) ExpectedError(
	t *testing.T,
	expected gateway.HTTPError,
) CallInfo[TRequest, TResponse] {
	t.Helper()
	c.ExpectedBody(t, expected.StatusCode(), expected)

	return c
}

func (c CallInfo[TRequest, TResponse]) ExpectedBody(t *testing.T, statusCode int, expected any) {
	t.Helper()

	var response map[string]any

	err := json.Unmarshal(c.body, &response)
	if err != nil {
		t.Fatal("Error unmarshal response", err, "\nResponse body: \n"+string(c.body))
	}

	must.Eq(t, c.response.StatusCode, statusCode, must.Sprintf("body message %s", c.body))

	expectedResponseJson, err := json.Marshal(expected)
	if err != nil {
		t.Fatal("Error marshal expected response", err)
	}

	var expectedJSON map[string]any

	err = json.Unmarshal(expectedResponseJson, &expectedJSON)
	if err != nil {
		t.Fatal("Error unmarshal response", err, "\nResponse body: \n"+string(c.body))
	}

	must.MapEq(t, expectedJSON, response)
}

func (c CallInfo[TRequest, TResponse]) ExpectedSuccess(t *testing.T, code int, expected TResponse) {
	t.Helper()

	var response TResponse

	err := json.Unmarshal(c.body, &response)
	if err != nil {
		t.Fatal("Error unmarshal response", err, "\nResponse body: \n"+string(c.body))
	}

	must.Eq(t, c.response.StatusCode, code, must.Sprintf("body message %s", c.body))
	must.Eq(t, expected, response)
}

type ConfigurableRoute[TContext any, TRequest any, TResult any] interface {
	ConfigureControllerMethod(m func(TContext, TRequest) (TResult, gateway.HTTPError)) gateway.Route
}

func TestAPICall[Controller controller, Context any, Request any, Response any](
	t *testing.T,
	ctr Controller,
	f func(Context, Request) (Response, gateway.HTTPError),
	cfg RequestConfiguration,
) CallInfo[Request, Response] {
	// ctr := h.Constructor()
	r, err := ctr.RouteTable().Find(cfg.routeStr)
	if err != nil {
		t.Fatal("Error while searching for rule. Original error: ", err.Error(),
			"\nList of registered routes for controller:\n",
			strings.Join(ctr.RouteTable().RouteStrings(), "\n"))
	}

	var (
		actualRequestObj  Request
		actualResponseObj Response
		actualError       gateway.HTTPError
	)

	if a, ok := r.(ConfigurableRoute[Context, Request, Response]); ok {
		r = a.ConfigureControllerMethod(func(ctx Context, req Request) (Response, gateway.HTTPError) {
			actualRequestObj = req
			resp, httpError := f(ctx, req)
			actualError = httpError
			actualResponseObj = resp

			return resp, httpError
		})
	} else {
		var f func(Context, Request) (Response, gateway.HTTPError)

		t.Fatal("Route is not configurable for ", reflect.TypeOf(f).String(),
			". Maybe wrong controller method signature.",
			" It can happens if you have changed a route without changing the expected method.")
	}

	handler := r.Handler()

	req := httptest.NewRequest(r.Method(), r.Path(), nil) //nolint:noctx // test helper, request context is irrelevant here
	for _, param := range cfg.configurations {
		err := param.apply(req)
		if err != nil {
			t.Fatal("Error while applying configuration. Original error: ", err.Error())
		}
	}

	rr := httptest.NewRecorder()
	handler(rr, req)

	return CallInfo[Request, Response]{
		request:        actualRequestObj,
		actualResponse: actualResponseObj,
		actualError:    actualError,
		response:       rr.Result(),
		body:           rr.Body.Bytes(),
		doc:            cfg.doc,
	}
}

func (h GroupRouteTestRunner[Controller]) Case(
	name string,
	test func(t *testing.T,
		ctr Controller,
		c RequestConfiguration,
	),
) GroupRouteTestRunner[Controller] {
	h.t.Run(name, func(t *testing.T) {
		t.Parallel()
		test(t, h.Constructor(), RequestConfiguration{
			routeStr:       h.RouteStr,
			doc:            h.doc,
			configurations: nil,
		})
	})

	return h
}

type RouteDocumentationDetails interface {
	AddOperationDetails(doc *openapi.Operation)
}

func (h GroupRouteTestRunner[Controller]) Documentation(
	doc *openapi.Documentation,
	details ...RouteDocumentationDetails,
) GroupRouteTestRunner[Controller] {
	doc.Capture()
	h.t.Cleanup(func() {
		doc.Release()
	})

	r, err := h.Constructor().RouteTable().Find(h.RouteStr)
	if err != nil {
		h.t.Fatal("Error while searching for rule. Original error: ", err.Error(),
			"\nList of registered routes for controller:\n",
			strings.Join(h.Constructor().RouteTable().RouteStrings(), "\n"))
	}

	path := doc.AddRoute(r.Path())

	operation, err := path.AddOperation(r.Method())
	if err != nil {
		h.t.Fatal("Error while adding operation to path. Original error: ", err.Error())
	}

	for _, d := range details {
		d.AddOperationDetails(operation)
	}

	h.doc = operation

	return h
}
