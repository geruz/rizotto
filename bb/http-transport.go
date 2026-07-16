package bb

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/geruz/rizotto/logger"
	"github.com/geruz/rizotto/validators"
	"github.com/go-chi/chi"
	"github.com/goccy/go-json"
)

func makeHTTPHandler[TRequest any, TAnswer any, TError ServiceError](
	uri string,
	handler func(context.Context, TRequest) (TAnswer, TError),
) func(ctx context.Context, i any) (any, ServiceError) {
	logParams := logger.KV("uri", uri) + " " + logger.KV("transport", "http")

	reqType := reflect.TypeFor[*TRequest]()

	return func(ctx context.Context, req any) (any, ServiceError) {
		// ctx, span := trace.Span(ctx, uri)
		// span.SetAttributes(trace.String("uri", uri))
		// defer span.End()
		requestsCountTotal.WithLabelValues(uri).Inc()

		defer func(start time.Time) {
			duration := time.Since(start)
			requestDuration.WithLabelValues(uri).Observe(duration.Seconds())
			logger.Trace(ctx, "Call service", logParams, logger.KVi("duration", duration.Milliseconds()))
		}(time.Now())

		requestPtr, ok := req.(*TRequest)

		if !ok {
			logger.Error(
				ctx,
				"Failed to cast request ",
				nil,
				reflect.TypeOf(req).Name()+"->"+reqType.String(),
				logger.KVj("request", req),
			)

			return nil, NewInvalidRequestError("invalid request type: " + reflect.TypeOf(req).Name() + "->" + reqType.String())
		}
		validateError := validators.ValidateHTTPRequest(ctx, req)
		if validateError != nil {
			jsonData, err := json.Marshal(validateError)
			if err != nil {
				logger.Error(ctx, "failed to marshal validation error", err, logger.KV("uri", uri))
			}

			logger.Warn(ctx, "validation error", logger.KV("uri", uri), logger.KV("error", string(jsonData)))

			return nil, NewValidationError("validation_error", string(jsonData))
		}

		return handler(ctx, *requestPtr)
	}
}

func makeHTTPEventHandler[TRequest any](
	handler func(context.Context, TRequest) bool,
) func(ctx context.Context, i any) bool {
	return func(ctx context.Context, req any) bool {
		request, ok := req.(*TRequest)
		if !ok {
			return false
		}

		return handler(ctx, *request)
	}
}

func StartHTTPEndpoint(ctx context.Context, port string) {
	router := chi.NewRouter()

	for uri, handlerParams := range callbacks {
		u, _ := url.Parse(uri)
		if u.Scheme != "http" {
			continue
		}

		registerRPCRoute(router, u, handlerParams)
		logger.Info(ctx, "Registering service endpoint: "+u.Path)
	}

	for uri, handlerParams := range events {
		u, _ := url.Parse(uri)
		if u.Scheme != "http" {
			continue
		}

		registerEventRoute(router, u, handlerParams)
		logger.Info(ctx, "Registering service endpoint: "+u.Path)
	}

	wrongMethod := func(w http.ResponseWriter, r *http.Request) {
		err := NewNotImplementedError("-", r.URL.Path)
		logger.Error(ctx, "Requested method not implemented", err, logger.KV("method", r.URL.Path))
		renderAnswer(ctx, w, err, nil)
	}
	router.Post("/*", wrongMethod)
	router.Get("/*", wrongMethod)
	router.Options("/*", wrongMethod)
	router.Patch("/*", wrongMethod)
	router.Delete("/*", wrongMethod)
	router.Put("/*", wrongMethod)

	go func() {
		logger.Info(ctx, "Starting services http endpoint on port "+port)

		const readHeaderTimeout = 10 * time.Second

		srv := &http.Server{ //nolint:exhaustruct
			Addr:              ":" + port,
			Handler:           router,
			ReadHeaderTimeout: readHeaderTimeout,
		}

		err := srv.ListenAndServe()
		if err != nil {
			panic(err)
		}
	}()
}

type CustomHeaders string

func GetCustomHeaders(ctx context.Context) map[string]string {
	headers, ok := ctx.Value(CustomHeaders("customHeaders")).(map[string]string)
	if !ok {
		return nil
	}

	return headers
}

func patchContextByCustomHeaders(ctx context.Context, r *http.Request) context.Context {
	headers := r.Header
	customHeaders := make(map[string]string)

	for k, v := range headers {
		lower := strings.ToLower(k)
		if strings.HasPrefix(lower, "x-") {
			customHeaders[lower] = v[0]
		}
	}

	return context.WithValue(ctx, CustomHeaders("customHeaders"), customHeaders)
}

func registerRPCRoute(router *chi.Mux, u *url.URL, params handlerRPCParams) {
	router.Post(u.Path, func(writer http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		reqObj := params.newRequest()

		body, err := io.ReadAll(req.Body)
		if err != nil {
			logger.Error(ctx, "Failed to read request body", err, logger.KV("method", req.URL.Path))
			renderAnswer(ctx, writer, err, nil)

			return
		}

		err = json.Unmarshal(body, reqObj)
		if err != nil {
			logger.Error(ctx, "Failed to decode request", err,
				logger.KV("method", req.URL.Path), logger.KV("body", string(body)))
			renderAnswer(ctx, writer, err, nil)

			return
		}

		ctxWithHTTPHeaders := patchContextByCustomHeaders(ctx, req)
		res, err := params.handler(ctxWithHTTPHeaders, reqObj)

		writer.Header().Set("Content-Type", "application/json")
		renderAnswer(ctxWithHTTPHeaders, writer, err, res)
	})
}

type Processed struct {
	Processed bool `json:"processed"`
}

func registerEventRoute(router *chi.Mux, u *url.URL, params handlerEventParams) {
	router.Post(u.Path, func(writer http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		reqObj := params.newRequest()

		err := json.NewDecoder(req.Body).Decode(reqObj)
		if err != nil {
			logger.Error(ctx, "Failed to decode request", err, logger.KV("method", req.URL.Path))
			renderAnswer(ctx, writer, err, nil)

			return
		}

		isProcessed := params.handler(ctx, reqObj)

		writer.Header().Set("Content-Type", "application/json")
		renderAnswer(ctx, writer, err, Processed{
			Processed: isProcessed,
		})
	})
}

func renderAnswer(ctx context.Context, writer http.ResponseWriter, err error, res any) struct{} {
	if err != nil {
		{
			var e *NotFoundError
			var e1 *InternalError
			var e2 *ValidationError
			var e3 *NotImplementedError
			var e4 *UnauthorizedError

			switch {
			case errors.As(err, &e):
				return renderError(ctx, e.baseError, e, writer)
			case errors.As(err, &e1):
				return renderError(ctx, e1.baseError, e1, writer)
			case errors.As(err, &e2):
				return renderError(ctx, e2.baseError, e2, writer)
			case errors.As(err, &e3):
				return renderError(ctx, e3.baseError, e3, writer)
			case errors.As(err, &e4):
				return renderError(ctx, e4.baseError, e4, writer)
			default:
				unwrappedErr := errors.Unwrap(err)
				if unwrappedErr != nil {
					return renderAnswer(ctx, writer, NewInternalError("Raw error", unwrappedErr), nil)
				}

				return renderAnswer(ctx, writer, NewInternalError("Raw error", e), nil)
			}
		}
	}

	data, err := json.MarshalContext(ctx, res)
	if err != nil {
		return renderAnswer(ctx, writer, NewInternalError("Failed to marshal response", err), nil)
	}

	writer.Header().Set("Content-Type", "application/json")
	_, err = writer.Write(data)
	logger.ErrorIfExists(ctx, err)

	return struct{}{}
}

func renderError(ctx context.Context, e baseError, payload any, w http.ResponseWriter) struct{} {
	data, err := json.Marshal(payload)

	w.Header().Set("Content-Type", "application/json")
	http.Error(w, string(data), e.StatusCode)
	logger.ErrorIfExists(ctx, err)

	return struct{}{}
}
