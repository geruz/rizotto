package main

import (
	"context"

	"github.com/geruz/rizotto"
	"github.com/geruz/rizotto/example/api/svg"
	api "github.com/geruz/rizotto/example/api/user-api"
	"github.com/geruz/rizotto/gateway"
	"github.com/geruz/rizotto/settings/env"
)

func main() {
	ctx := context.Background()
	rizotto.MustInitEnv(ctx)
	rizotto.MustInitLogger(ctx)

	rizotto.InitMetrics(ctx, ":"+env.GetStringValue("METRICS_PORT", "9091"))
	// rizotto.InitTracer(ctx)
	// rizotto.InitScheduler(ctx)
	gt := gateway.NewHTTPGateway()

	err := gt.Routing(
		gateway.JoinRouteTables(
			api.NewUserController().RouteTable(),
			svg.NewSVGController().RouteTable(),
		),
	).ListenAndServe(ctx, ":"+env.GetStringValue("PORT", "9090"))
	if err != nil {
		panic(err)
	}
}
