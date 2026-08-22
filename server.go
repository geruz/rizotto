package rizotto

import (
	"context"

	"github.com/geruz/rizotto/gateway"
	"github.com/geruz/rizotto/logger"
	"github.com/geruz/rizotto/metrics"
	"github.com/geruz/rizotto/settings/env"
	"github.com/geruz/rizotto/trace"
)

var MustInitMetrics = metrics.MustInitMetrics
var MustInitTracer = trace.MustInit

type HTTPContext = gateway.HTTPContext
type Controller = gateway.Controller

func MustInitEnv(ctx context.Context) {
	env.MustLoadEnvFile(".env")
}

func MustInitLogger(ctx context.Context) {
	logger.MustInit()
}

func InitScheduler(ctx context.Context) {
}
