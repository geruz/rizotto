package bb

import (
	"context"
	"errors"
	"net/url"

	"github.com/geruz/rizotto/logger"
	"github.com/geruz/rizotto/metrics"
)

const (
	natsProtocol string = "nats"
	httpProtocol string = "http"
)

type handlerRPCParams struct {
	newRequest func() any
	handler    func(context.Context, any) (any, ServiceError)
}
type handlerEventParams struct {
	newRequest func() any
	handler    func(context.Context, any) bool
}

var (
	callbacks = map[string]handlerRPCParams{}
	events    = map[string]handlerEventParams{}
)

var requestsCountTotal = metrics.CounterVec(
	"service_requests_total",
	"Current service requests count.",
	[]string{"uri"},
)

var requestDuration = metrics.SummaryVec(
	"service_requests_durations_seconds",
	"Service requests durations in seconds.",
	metrics.Percentiles_60_90_99,
	[]string{"uri"},
)

func mustGetScheme(uri string) string {
	u, err := url.Parse(uri)
	if err != nil {
		panic(err)
	}

	return u.Scheme
}

func classifyRPCError(ctx context.Context, uri, service, method string, err ServiceError) ServiceError {
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

func MustBind[
	T_RPC ~func(context.Context, TRequest) (TAnswer, ServiceError),
	TRequest any,
	TAnswer any,
](
	uri string,
) T_RPC {
	parts, err := url.Parse(uri)
	if err != nil {
		panic(err)
	}

	service := parts.Host
	method := parts.Path

	return T_RPC(func(ctx context.Context, req TRequest) (TAnswer, ServiceError) {
		params, ok := callbacks[uri]
		if !ok {
			logger.Error(ctx, "method not registered", nil, logger.KV("uri", uri))
			var answer TAnswer

			return answer, NewNotImplementedError(service, method)
		}

		res, err := params.handler(ctx, &req)
		if err != nil {
			var answer TAnswer

			return answer, classifyRPCError(ctx, uri, service, method, err)
		}

		answer, ok := res.(TAnswer)
		if !ok {
			var answer TAnswer

			return answer, NewWrongContractError(service, method)
		}

		return answer, nil
	})
}

func MustRegister[TRequest any, TAnswer any, TError ServiceError](
	uri string,
	handler func(context.Context, TRequest) (TAnswer, TError),
) {
	scheme := mustGetScheme(uri)
	switch scheme {
	case natsProtocol:
		panic("nats not implemented for RPC")
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
	case httpProtocol:
		events[uri] = newHandlerEventParams[TRequest](makeHTTPEventHandler(handler))
	default:
		panic("Unknown scheme: " + scheme)
	}
}
