package logger

import (
	"context"
	"encoding/json"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/geruz/rizotto/meta"
	"github.com/geruz/rizotto/settings/env"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type LogLevel string

const (
	TraceLevel LogLevel = "TRACE"
	DebugLevel LogLevel = "DEBUG"
	InfoLevel  LogLevel = "INFO"
	WarnLevel  LogLevel = "WARN"
	ErrorLevel LogLevel = "ERROR"
	FatalLevel LogLevel = "FATAL"
)

var minLevel int

const (
	AllLevelsIndex  = -1
	TraceLevelIndex = 0
	DebugLevelIndex = 1
	InfoLevelIndex  = 2
	WarnLevelIndex  = 3
	ErrorLevelIndex = 4
	FatalLevelIndex = 5
)

func levelIndex(l LogLevel) int {
	switch l {
	case TraceLevel:
		return TraceLevelIndex
	case DebugLevel:
		return DebugLevelIndex
	case InfoLevel:
		return InfoLevelIndex
	case WarnLevel:
		return WarnLevelIndex
	case ErrorLevel:
		return ErrorLevelIndex
	case FatalLevel:
		return FatalLevelIndex
	default:
		return AllLevelsIndex
	}
}

func createLogChecker(level LogLevel) func(ctx context.Context) bool {
	levelIndex := levelIndex(level)

	return func(ctx context.Context) bool {
		return levelIndex >= getLogLevel(ctx)
	}
}

func MustInit() {
	minLevel = levelIndex(env.MustGetEnumValue("LOG_LEVEL", []LogLevel{
		TraceLevel,
		DebugLevel,
		InfoLevel,
		WarnLevel,
		ErrorLevel,
		FatalLevel,
	}))
}

func KV[TValue ~string](key string, value TValue) string {
	return key + "=" + string(value)
}

func KVj[TValue any](key string, value TValue) string {
	json, err := json.Marshal(value)
	if err != nil {
		return key + "=marshal error"
	}

	return key + "=" + string(json)
}

func KVi[TValue ~int64 | ~int | ~int32](key string, value TValue) string {
	return key + "=" + strconv.FormatInt(int64(value), 10)
}

func KVb[TValue ~bool](key string, value TValue) string {
	return key + "=" + strconv.FormatBool(bool(value))
}

func KVf[TValue ~float32 | ~float64](key string, value TValue) string {
	return key + "=" + strconv.FormatFloat(float64(value), 'f', -1, 32)
}

func currentTime() string {
	return "[" + time.Now().Format("2006-01-02 15:04:05,000") + "]"
}

func traceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span == nil || !span.SpanContext().IsSampled() {
		return ""
	}

	return "traceID=" + span.SpanContext().TraceID().String()
}

func fromCtx(ctx context.Context) (string, string) {
	if ctx == nil {
		return "", ""
	}

	left := ""
	right := ""
	firstIterFroRight := true

	var (
		leftSb108  strings.Builder
		rightSb108 strings.Builder
	)

	for k, v := range meta.GetMeta(ctx) {
		if k == "namespace" {
			leftSb108.WriteString("[")
			leftSb108.WriteString(v)
			leftSb108.WriteString("] ")
		} else {
			if !firstIterFroRight {
				rightSb108.WriteString(" ")
			}

			rightSb108.WriteString(k)
			rightSb108.WriteString("=")
			rightSb108.WriteString(v)

			firstIterFroRight = false
		}
	}

	left += leftSb108.String()
	right += rightSb108.String()

	return left, right
}

var levelPrefix = map[LogLevel]string{
	TraceLevel: " [TRACE] ",
	DebugLevel: " [DEBUG] ",
	InfoLevel:  " [INFO]  ",
	WarnLevel:  " [WARN]  ",
	ErrorLevel: " [ERROR] ",
	FatalLevel: " [FATAL] ",
}

func format(ctx context.Context, lp string, message string, messages ...string) string {
	left, right := fromCtx(ctx)

	l := currentTime() + lp + left + message

	var lSb134 strings.Builder
	for _, message := range messages {
		lSb134.WriteString(" ")
		lSb134.WriteString(message)
	}

	l += lSb134.String()

	traceID := traceID(ctx)
	if traceID != "" {
		l += " " + traceID
	}

	if right != "" {
		l += " " + right
	}

	return l
}

func mustCreatePrintLog(level LogLevel) func(ctx context.Context, message string, messages ...string) {
	enableLog := createLogChecker(level)

	lp, ok := levelPrefix[level]
	if !ok {
		panic("unknown log level: " + string(level))
	}

	return func(ctx context.Context, message string, messages ...string) {
		if !enableLog(ctx) {
			return
		}

		l := format(ctx, lp, message, messages...)
		println(l) //nolint:forbidigo
	}
}

var (
	Trace = mustCreatePrintLog(TraceLevel)
	Debug = mustCreatePrintLog(DebugLevel)
	Info  = mustCreatePrintLog(InfoLevel)
	Warn  = mustCreatePrintLog(WarnLevel)
	Fatal = mustCreatePrintLog(FatalLevel)
)

func formatError(ctx context.Context, description string, err error, kvs ...string) string {
	text := description
	if err != nil {
		text += ": " + strings.ReplaceAll(err.Error(), "ERROR:", "")
	} else {
		text += ""
	}

	if text == "" {
		text = "Unknown error"
	}

	if len(kvs) > 0 {
		text += " "
		text += strings.Join(kvs, " ")
	}

	span := trace.SpanFromContext(ctx)
	if span != nil {
		span.SetStatus(codes.Error, text)
	}

	formattedLine := format(ctx, levelPrefix[ErrorLevel], text)
	const BufferSize = 1024 * 4
	buf := make([]byte, BufferSize)
	runtime.Stack(buf, false)

	return formattedLine + "\n" + string(buf)
}

func Error(ctx context.Context, description string, err error, kvs ...string) {
	println(formatError(ctx, description, err, kvs...)) //nolint:forbidigo
}

func ErrorText(ctx context.Context, message string, kvs ...string) {
	Error(ctx, message, nil, kvs...)
}

func ErrorIfExists(ctx context.Context, err error) {
	if err != nil {
		Error(ctx, "", err)
	}
}

type ctxKV struct{}

func WithParams(ctx context.Context, params map[string]string) context.Context {
	return context.WithValue(ctx, ctxKV{}, lo.Assign(
		getAdditionalInfo(ctx),
		params,
	))
}
func getAdditionalInfo(ctx context.Context) map[string]string {
	if ctx == nil {
		return nil
	}
	if kvs, ok := ctx.Value(ctxKV{}).(map[string]string); ok {
		return kvs
	}

	return nil
}
