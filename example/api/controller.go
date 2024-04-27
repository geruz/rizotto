package api

import "github.com/geruz/rizotto"

func PrivateArea(ctx rizotto.HTTPContext) UserHTTPContext {
	return UserHTTPContext{
		HTTPContext: ctx,
		User: CurrentUser{
			Name: "John",
		},
	}
}

func PublicArea(ctx rizotto.HTTPContext) PublicHTTPContext {
	return PublicHTTPContext{
		HTTPContext: ctx,
	}
}

type CurrentUser struct {
	Name string
}

type PublicHTTPContext struct {
	rizotto.HTTPContext
}
type UserHTTPContext struct {
	rizotto.HTTPContext

	User CurrentUser
}
