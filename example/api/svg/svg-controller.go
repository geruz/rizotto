package svg

import (
	"text/template"

	"github.com/geruz/rizotto"
	"github.com/geruz/rizotto/async"
	"github.com/geruz/rizotto/content"
	tr "github.com/geruz/rizotto/example/services/templates"
	"github.com/geruz/rizotto/example/services/templates/templates-client"
	"github.com/geruz/rizotto/gateway"
	"github.com/geruz/rizotto/logger"
)

type SVGController struct {
	rizotto.Controller

	tmplSrv interface {
		templates.GetTemplate
		templates.GetModel
	}
}

func noAuth(ctx gateway.HTTPContext) (gateway.HTTPContext, gateway.HTTPError) {
	return ctx, nil
}

func NewSVGController() SVGController {
	ctrl := SVGController{
		Controller: rizotto.Controller{},
		tmplSrv:    tr.BindService(),
	}
	ctrl.AddRoutes(
		gateway.Content("GET /svg/{user}/{template}/{key}", noAuth, ctrl.renderSVG),
	)

	return ctrl
}

type RenderTemplateRequest struct {
	User     string `in:"path=user"`
	Template string `in:"path=template"`
	Key      string `in:"path=key"`
}

func (ctrl SVGController) renderSVG(
	ctx gateway.HTTPContext, req RenderTemplateRequest,
) (SVGTemplate, gateway.HTTPError) {
	const exampleId = 12
	tmpl, model, err := async.Parallel2(

		async.Async1(ctrl.tmplSrv.GetTemplate, templates.GetTemplateRequest{ID: exampleId}),
		async.Async1(ctrl.tmplSrv.GetModel, templates.ModelRequest{
			User:     req.User,
			Template: req.Template,
		}),
	)(ctx)
	if err != nil {
		logger.Error(ctx, "Failed to parse template", err)

		return SVGTemplate{SVGContentType: content.SVGContentType{}, Template: content.Template{}}, gateway.NewInternalError()
	}

	tmp2, err := template.New("svg").Parse(tmpl.Template)
	if err != nil {
		logger.Error(ctx, "Failed to render template", err)

		return SVGTemplate{SVGContentType: content.SVGContentType{}, Template: content.Template{}}, gateway.NewInternalError()
	}

	return SVGTemplate{
		SVGContentType: content.SVGContentType{},
		Template:       content.NewTemplate(tmp2, model),
	}, nil
}

type SVGTemplate struct {
	content.SVGContentType
	content.Template
}
