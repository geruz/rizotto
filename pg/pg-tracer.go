package pg

import (
	"context"
	"strings"
	"time"

	"github.com/geruz/rizotto/logger"
	"github.com/geruz/rizotto/trace"
	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type (
	pgTracer  struct{}
	SQLParams struct {
		SQL          string
		Name         string
		StartTime    time.Time
		Error        error
		RowsAffected int64
		Duration     time.Duration
	}
	sqlContextKey struct{}
)

func (pl pgTracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	params := SQLParams{
		SQL:          data.SQL,
		Name:         pl.getSQLName(data.SQL),
		StartTime:    time.Now(),
		Error:        nil,
		RowsAffected: 0,
		Duration:     0,
	}
	ctx = context.WithValue(ctx, sqlContextKey{}, params)
	ctx, span := trace.Span(ctx, "sql "+params.Name)
	span.SetAttributes(trace.String("name", params.Name))

	return ctx
}

func (pl pgTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	params, ok := ctx.Value(sqlContextKey{}).(SQLParams)
	if !ok {
		return
	}
	span := trace.SpanFromContext(ctx)
	defer span.End()
	if data.Err != nil {
		span.RecordError(data.Err)
	}

	params.Error = data.Err
	params.RowsAffected = data.CommandTag.RowsAffected()
	params.Duration = time.Since(params.StartTime)
	sqlQueryDuration.Record(ctx, params.Duration.Seconds(), metric.WithAttributes(attribute.String("name", params.Name)))

	pl.log(ctx, params)
}

func (pl pgTracer) log(ctx context.Context, params SQLParams) {
	logBody := []string{
		logger.KV("name", params.Name),
		logger.KV("duration", params.Duration.String()),
		logger.KVi("rows", params.RowsAffected),
	}
	err := params.Error
	logger.Trace(ctx, "sql body: "+params.SQL, logBody...)
	if err == nil {
		return
	}
	if IsNotFoundError(err) || IsDuplicateError(err) {
		logger.Warn(ctx, err.Error(), logBody...)
	} else {
		logger.Error(ctx, "SQL error ", err, logBody...)
	}
}

func (pl pgTracer) getSQLName(query string) string {
	if query == "" {
		return "unknown"
	}
	firstLine := strings.Split(query, "\n")[0]
	lines := strings.SplitN(firstLine, "name: ", 2)
	if len(lines) != 2 {
		return "unknown"
	}

	return strings.Split(lines[1], " ")[0]
}
