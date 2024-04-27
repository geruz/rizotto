package content

import (
	"io"
	"text/template"
)

type Template struct {
	tmpl *template.Template
	data any
}

func NewTemplate(tmpl *template.Template, data any) Template {
	return Template{
		tmpl: tmpl,
		data: data,
	}
}

func (t Template) Write(bw io.Writer) error {
	return t.tmpl.Execute(bw, t.data)
}
