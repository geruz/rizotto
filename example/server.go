package main

import (
	"context"

	"github.com/geruz/rizotto"
	"github.com/geruz/rizotto/example/api/svg"
	api "github.com/geruz/rizotto/example/api/user-api"
	usersrv "github.com/geruz/rizotto/example/services/users"
	"github.com/geruz/rizotto/example/services/users/user-client"
	"github.com/geruz/rizotto/gateway"
	"github.com/geruz/rizotto/pg"
	"github.com/geruz/rizotto/settings/env"
)

func main() {
	ctx := context.Background()
	rizotto.MustInitEnv(ctx)
	rizotto.MustInitLogger(ctx)

	rizotto.InitMetrics(ctx, ":"+env.GetStringValue("METRICS_PORT", "9091"))
	rizotto.MustInitTracer(ctx, map[string]string{
		"service.name":    env.MustGetStringValue("SERVICE_NAME"),
		"service.version": env.MustGetStringValue("SERVICE_VERSION"),
		"environment":     env.MustGetStringValue("ENVIRONMENT"),
	})
	// rizotto.InitScheduler(ctx)
	gt := gateway.NewHTTPGateway()

	pgPool := pg.MustOpenConnection(ctx, env.MustGetStringValue("DATABASE_URL"))

	user.RegisterServer(
		usersrv.NewUserService(pgPool),
	)
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
