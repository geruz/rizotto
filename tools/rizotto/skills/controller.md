# Controllers

A controller is the HTTP face of one service: it declares routes, parses and
validates requests, calls the RPCs of the service client and maps service errors
onto HTTP errors. It lives in its own package so that several controllers can
coexist:

    api/controller.go        the HTTP contexts shared by every controller
    api/order-api/order.go   the Order controller (package orderapi)
    api/order-api/order_test.go  its route tests, which also write openapi.yml

## 1. Declaring routes

```go
type OrderController struct {
    gateway.Controller

    getOrderRPC     order.GetOrderRPC     // the RPC function types from bind-gen.go
    selectOrdersRPC order.SelectOrdersRPC
}

func NewOrderController() OrderController {
    client := order.BindClient()
    ctrl := OrderController{
        Controller:      rizotto.Controller{},
        getOrderRPC:     client.GetOrderRPC,
        selectOrdersRPC: client.SelectOrdersRPC,
    }
    ctrl.AddRoutes(
        gateway.JSONMethod("GET /api/v1/orders/{order_id}", api.PrivateArea, ctrl.getOrder),
        gateway.JSONMethod("GET /api/v1/orders", api.PrivateArea, ctrl.listOrders),
    )

    return ctrl
}
```

- the route string is `"<METHOD> <path>"`; path variables use `{name}`;
- the RPCs are stored as fields, which is what makes them mockable in tests;
- `gateway.JSONMethod` renders the answer as JSON. `gateway.Content` renders a
  `ContentProvider` instead (SVG, templates, files);
- `AddRoutes` fills the embedded `gateway.Controller`, and `RouteTable()` hands the
  routes to the server.

## 2. Contexts: who may call the route

The second argument of a route builds the handler context from the request. Both
builders live in `api/controller.go`:

```go
func PrivateArea(ctx rizotto.HTTPContext) (UserHTTPContext, rizotto.HTTPError) // authenticated routes
func PublicArea(ctx rizotto.HTTPContext) (PublicHTTPContext, rizotto.HTTPError) // open routes
```

`UserHTTPContext` carries the current user; the scaffolded version returns a
placeholder — put the real authentication (token parsing, session lookup) there, and
every private route gets it. The context is also a `context.Context`, so it is passed
straight into the RPC call.

An area function answering an error rejects the request **before** the handler runs,
which is how a route answers 401 to a caller it cannot authenticate:

```go
func PrivateArea(ctx rizotto.HTTPContext) (UserHTTPContext, rizotto.HTTPError) {
    session, err := lookUpSession(ctx)
    if err != nil {
        return UserHTTPContext{}, gateway.NewUnauthorizedError("no session")
    }

    return UserHTTPContext{HTTPContext: ctx, User: CurrentUser{Name: session.Name}}, nil
}
```

## 3. The request object

One struct per route describes where each field comes from:

```go
type OrderRequest struct {
    // json:"-" keeps the json body from overwriting what the path carries
    OrderID int `in:"path=order_id" json:"-"`
}

type OrderListRequest struct {
    Offset int `in:"query=offset;default=0" json:"-"`
    Limit  int `in:"query=limit;default=10" json:"-"`
}

type CreateOrderRequest struct {
    Name string `json:"name" validate:"required"`
}

type UpdateOrderRequest struct {
    OrderID int    `in:"path=order_id" json:"-"`
    Name    string `json:"name"        validate:"required"`
}
```

- `in:"path=<name>"` reads a path variable, `in:"query=<name>"` a query parameter,
  `;default=<value>` gives it a default (a defaulted parameter is documented as
  optional);
- a field **without** an `in` tag is read from the JSON body — the body is
  unmarshalled into the same struct after the path and query values are filled in,
  which is why path and query fields carry `json:"-"`;
- `application/x-www-form-urlencoded` bodies are accepted as well;
- `validate:` tags are checked by go-playground/validator; a failure answers 400 with
  a field-by-field validation error, naming the fields by their `json` tag.

## 4. The handler

```go
func (ctrl OrderController) getOrder(
    ctx api.UserHTTPContext, req OrderRequest,
) (Order, gateway.HTTPError) {
    found, err := ctrl.getOrderRPC(ctx, order.GetOrderRequest{OrderID: req.OrderID})
    if err != nil {
        return Order{ID: 0, Name: ""}, gateway.MapError(err).
            IfNotFound(orderNotFound).
            Others(internalError)
    }

    return ServiceOrderToAPIOrder(found, 0), nil
}
```

The signature is always `func(ctx <Context>, req <Request>) (<Response>, gateway.HTTPError)`.
The response struct is marshalled to JSON with its `json` tags — keep it separate
from the service type so the HTTP contract does not follow internal changes.

### Errors

Declare the HTTP errors of the controller once and map the service error onto them:

```go
var (
    orderNotFound = gateway.NewNotFoundError(
        gateway.ErrorCode("order_not_found"), "Order not found", "Order with the given id was not found",
    )
    internalError = gateway.NewInternalError()
)

gateway.MapError(err).IfNotFound(orderNotFound).Others(internalError)
```

`gateway.NewInvalidRequest(msg)` answers 400, `NewValidationError()` collects field
errors. The body of an error is `{"code", "message", "errorCode", "details"}`.

## 5. Registering the controller

In `server.go`:

```go
import orderapi "<module>/api/order-api"

err := gt.Routing(
    gateway.JoinRouteTables(
        orderapi.NewOrderController().RouteTable(),
    ),
).ListenAndServe(ctx, ":"+env.GetStringValue("PORT", "9090"))
```

The constructor binds the service client, so the service must be registered earlier
in `main` (see `rizotto service skill`).

## 6. Route tests and the OpenAPI document

Route tests run the real routing stack — parsing, validation, rendering — with the
RPCs mocked, and they double as the source of `openapi.yml`:

```go
func Test_GetOrderRoute(t *testing.T) {
    t.Parallel()

    api.GroupRouteTests(t, "GET /api/v1/orders/{order_id}", NewOrderController).
        Documentation(doc.ApiPublicDocumentationV1, openapi.Description("Returns the `Order`.")).
        Case("existing order", testGetOrderCase).
        Case("order not found", testGetOrderNotFoundCase)
}

func testGetOrderCase(t *testing.T, ctr OrderController, cfg api.RequestConfiguration) {
    t.Helper()
    mock.RPC(&ctr.getOrderRPC).Success(order.Order{ID: 1, Name: "table"})

    api.TestAPICall(t, ctr, ctr.getOrder, cfg.WithPathVar("order_id", "1")).
        AddInDocumentation(t, "OK").
        ExpectedRequest(t, OrderRequest{OrderID: 1}).
        ExpectedResponse(t, Order{ID: 1, Name: "table"})
}
```

- `mock.RPC(&ctr.field).Success(v)` / `.Error(bb.NewNotFoundError("…"))` replaces the
  bound RPC; `.CallsCount()` and `.Call(n)` inspect what it received;
- the request is configured with `WithPathVar`, `WithQuery("limit=10&offset=2")` and
  `WithJSONBody(payload)`;
- `AddInDocumentation(t, "OK")` records the request and the answer as an example in
  the document; `ExpectedResponse` asserts a 200 body, `ExpectedError(t, orderNotFound)`
  asserts the mapped error, `ExpectedAnswer(t, code, body)` any other answer;
- when the last group releases the document, it is written to `openapi.yml` next to
  the test — `task test` keeps it up to date.

## Adding one

    rizotto controller add       # asks for the service, the routes, the area and the prefix

It generates the controller and its route tests for the CRUD methods the service
implements, and prints the route table to join in `server.go`.
