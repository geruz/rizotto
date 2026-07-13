package main

import (
	"context"

	"github.com/geruz/rizotto"
	"github.com/geruz/rizotto/example/api/svg"
	api "github.com/geruz/rizotto/example/api/user-api"
	"github.com/geruz/rizotto/settings/env"
)

func main() {
	ctx := context.Background()
	rizotto.MustInitEnv(ctx)
	rizotto.MustInitLogger(ctx)

	rizotto.InitMetrics(ctx, ":8081")
	// rizotto.InitTracer(ctx)
	// rizotto.InitScheduler(ctx)
	// MustBindServices(ctx)
	gateway := rizotto.NewHTTPGateway()

	err := gateway.Routing(
		rizotto.JoinRouteTables(
			api.NewUserController().RouteTable(),
			svg.NewSVGController().RouteTable(),
		),
	).ListenAndServe(ctx, ":"+env.GetStringValue("PORT", "9090"))
	if err != nil {
		panic(err)
	}
}
