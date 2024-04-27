package bb

/*
import (
	"context"
	"net/url"
	"time"

)

func makeRabbitEventHandler[TRequest any](
	uri string,
	handler func(context.Context, TRequest) bool,
) func(ctx context.Context, i interface{}) bool {
	logParams := logger.KV("uri", uri) + " " + logger.KV("transport", "rabbitmq")
	return func(ctx context.Context, i interface{}) bool {
		requestsCountTotal.WithLabelValues(uri).Inc()
		defer func(start time.Time) {
			duration := time.Since(start)
			requestDuration.WithLabelValues(uri).Observe(duration.Seconds())
			logger.Trace(ctx, "Call service", logParams, logger.KVi("duration", duration.Milliseconds()))
		}(time.Now())
		logger.Info(ctx, "Call internal service", logger.KV("uri", uri))
		request, ok := i.(*TRequest)
		if !ok {
			logger.Error(ctx, "Failed to cast request", nil, logParams)
			return false
		}
		return handler(ctx, *request)
	}
}

func StartRabbitTransport(ctx context.Context, getConnection rabbitmq.ConnectionProvider) {
	for uri, handlerParams := range events {
		urlParts, _ := url.Parse(uri)
		if urlParts.Scheme != "rabbitmq" {
			continue
		}
		workerCount := utils.ParseIntOrDefault(urlParts.Query().Get("workers_count"), 10)
		logger.Info(ctx, "Registering service rabbit consumer: "+urlParts.Path)
		action := urlParts.Path[1:]
		consumer := rabbitmq.NewConsumer(
			getConnection,
			urlParts.Host,
			urlParts.Host+"/"+action,
			action,
		)
		consumer.Consume(ctx, handlerParams.newRequest, handlerParams.handler, workerCount)
	}
}
*/
