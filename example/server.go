package main

import (
	"context"

	"github.com/geruz/rizotto"
	"github.com/geruz/rizotto/example/api/svg"
	api "github.com/geruz/rizotto/example/api/user-api"
)

func main() {
	ctx := context.Background()
	rizotto.MustInitEnv(ctx)
	rizotto.MustInitLogger(ctx)

	// rizotto.InitMetrics(ctx, ":8081")
	// rizotto.InitTracer(ctx)
	// rizotto.InitScheduler(ctx)
	// MustBindServices(ctx)
	gateway := rizotto.NewHTTPGateway()

	err := gateway.Routing(
		rizotto.JoinRouteTables(
			api.NewUserController().RouteTable(),
			svg.NewSVGController().RouteTable(),
		),
	).ListenAndServe(ctx, ":9090")
	if err != nil {
		panic(err)
	}
}
