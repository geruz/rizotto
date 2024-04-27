package doc

import (
	"github.com/geruz/rizotto/documentation/openapi"
)

var ApiPublicDocumentationV1 = &openapi.Documentation{
	OpenAPI: "3.1.0",
	Info: openapi.Info{
		Version: "1.0.0",
		Title:   "Swagger Petstore - OpenAPI 3.1",
		Summary: "User API",
		Description: `This is a sample Pet Store Server based on the OpenAPI 3.1 specification.  You can find out more about
Swagger at [https://swagger.io](https://swagger.io). In the third iteration of the pet store, we've switched to
the design first approach! You can now help us improve the API whether it's by making changes to the definition
itself or to the code. That way, with time, we can improve the API in general, and expose some of the new
features in OAS3.`,
		TermsOfService: "http://swagger.io/terms/",
		Contract: &openapi.ContractInfo{
			Name:  "User API",
			Url:   "",
			Email: "apiteam@swagger.io",
		},
		License: &openapi.LicenseInfo{
			Name:       "Apache 2.0",
			Identifier: "",
			Url:        "http://www.apache.org/licenses/LICENSE-2.0.html",
		},
	},
	Servers: []openapi.ServerInfo{
		{
			Url:         "http://petstore.swagger.io/api",
			Description: "",
			Variables:   nil,
		},
	},
	Paths:    nil,
	Security: openapi.Security{SecuritySchemes: nil},
	Tags:     nil,
	ExternalDocs: &openapi.ExternalDocs{
		Description: "Find out more about Swagger",
		Url:         "http://swagger.io",
	},
}
