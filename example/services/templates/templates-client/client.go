package templates

import "context"

type GetTemplateRequest struct {
	ID int `json:"id"`
}
type Template struct {
	ID       int    `json:"id"`
	Template string `json:"template"`
}

type Model struct {
	ID   int `json:"id"`
	Data any `json:"data"`
}
type ModelRequest struct {
	User     string `json:"user"`
	Template string `json:"template"`
}

type GetTemplate interface {
	GetTemplate(ctx context.Context, req GetTemplateRequest) (Template, error)
}
type GetModel interface {
	GetModel(ctx context.Context, req ModelRequest) (Model, error)
}

type TemplateService interface {
	GetTemplate
	GetModel
}
