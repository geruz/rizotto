package api

import "github.com/geruz/rizotto"

func PrivateArea(ctx rizotto.HTTPContext) (UserHTTPContext, rizotto.HTTPError) {
	return UserHTTPContext{
		HTTPContext: ctx,
		User: CurrentUser{
			Name: "John",
		},
	}, nil
}

func PublicArea(ctx rizotto.HTTPContext) (PublicHTTPContext, rizotto.HTTPError) {
	return PublicHTTPContext{
		HTTPContext: ctx,
	}, nil
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
