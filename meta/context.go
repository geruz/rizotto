package meta

import (
	"context"

	"github.com/samber/lo"
)

type (
	meta struct{}
)

func GetMeta(ctx context.Context) map[string]string {
	if ctx == nil {
		return nil
	}

	if kvs, ok := ctx.Value(meta{}).(map[string]string); ok {
		return kvs
	}

	return nil
}

func WithParams(ctx context.Context, params map[string]string) context.Context {
	return context.WithValue(ctx, meta{}, lo.Assign(
		GetMeta(ctx),
		params,
	))
}
