package bb

import (
	"context"
	"errors"
	"net/url"

	"github.com/geruz/rizotto/logger"
	"github.com/geruz/rizotto/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	rabbitProtocol string = "rabbitmq"
	httpProtocol   string = "http"
)

type handlerRPCParams struct {
	newRequest func() any
	handler    func(context.Context, any) (any, error)
}
type handlerEventParams struct {
	newRequest func() any
	handler    func(context.Context, any) bool
}

var (
	callbacks = map[string]handlerRPCParams{}
	events    = map[string]handlerEventParams{}
)

var requestsCountTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace:   "",
		Subsystem:   "",
		Name:        "service_requests_total",
		Help:        "Current service requests count.",
		ConstLabels: nil,
	},
	[]string{"uri"},
)

var requestDuration = metrics.SummaryVec(
	"service_requests_durations_seconds",
	"Service requests durations in seconds.",
	metrics.Percentiles_60_90_99,
	[]string{"uri"},
)

func init() {
	prometheus.MustRegister(requestsCountTotal)
	prometheus.MustRegister(requestDuration)
}

func mustGetScheme(uri string) string {
	u, err := url.Parse(uri)
	if err != nil {
		panic(err)
	}

	return u.Scheme
}

var errWrongAnswerType = errors.New("wrong answer type")

func classifyRPCError(ctx context.Context, uri, service, method string, err error) error {
	var (
		notFound          *NotFoundError
		validationErr     *ValidationError
		internalErr       *InternalError
		notImplementedErr *NotImplementedError
		unauthorizedError *UnauthorizedError
	)

	switch {
	case errors.As(err, &notFound):
		logger.Warn(ctx, "not found error", err.Error(), logger.KV("uri", uri))
	case errors.As(err, &validationErr):
		logger.Warn(ctx, "validation error", err.Error(), logger.KV("uri", uri))
	case errors.As(err, &internalErr):
	case errors.As(err, &notImplementedErr):
		logger.Error(ctx, "not implemented error", err, logger.KV("uri", uri))
	case errors.As(err, &unauthorizedError):
		logger.Warn(ctx, "unauthorized error", err.Error(), logger.KV("uri", uri))
	default:
		logger.Error(ctx, "unknown error", err, logger.KV("uri", uri))

		return NewInternalError("unknown error from "+service+"/"+method, err)
	}

	return err
}

func MustBind[T_RPC ~func(context.Context, TRequest) (*TAnswer, error), TRequest any, TAnswer any](uri string) T_RPC {
	parts, err := url.Parse(uri)
	if err != nil {
		panic(err)
	}

	service := parts.Host
	method := parts.Path

	return T_RPC(func(ctx context.Context, req TRequest) (*TAnswer, error) {
		params, ok := callbacks[uri]
		if !ok {
			logger.Error(ctx, "method not registered", nil, logger.KV("uri", uri))

			return nil, NewNotImplementedError(service, method)
		}

		res, err := params.handler(ctx, &req)
		if err != nil {
			return nil, classifyRPCError(ctx, uri, service, method, err)
		}

		answer, ok := res.(*TAnswer)
		if !ok {
			return nil, errWrongAnswerType
		}

		return answer, nil
	})
}

func MustRegister[TRequest any, TAnswer any, TError error](
	uri string,
	handler func(context.Context, TRequest) (*TAnswer, TError),
) {
	scheme := mustGetScheme(uri)
	switch scheme {
	case rabbitProtocol:
		panic("rabbitmq not implemented for RPC")
	case httpProtocol:
		callbacks[uri] = handlerRPCParams{
			newRequest: func() any { return new(TRequest) },
			handler:    makeHTTPHandler(uri, handler),
		}
	default:
		panic("Unknown scheme: " + scheme)
	}
}

func newHandlerEventParams[TRequest any](handler func(context.Context, any) bool) handlerEventParams {
	return handlerEventParams{
		newRequest: func() any { return new(TRequest) },
		handler:    handler,
	}
}

func MustRegisterEvent[TRequest any](uri string, handler func(context.Context, TRequest) bool) {
	scheme := mustGetScheme(uri)
	switch scheme {
	// case rabbitProtocol:
	//	events[uri] = newHandlerEventParams[TRequest](makeRabbitEventHandler(uri, handler))
	case httpProtocol:
		events[uri] = newHandlerEventParams[TRequest](makeHTTPEventHandler(handler))
	default:
		panic("Unknown scheme: " + scheme)
	}
}
