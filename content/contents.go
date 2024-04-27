package content

type JSONContentType struct{}

func (t JSONContentType) ContentType() string {
	return "application/json"
}

type HTMLContentType struct{}

func (t HTMLContentType) ContentType() string {
	return "text/html"
}

type SVGContentType struct{}

func (t SVGContentType) ContentType() string {
	return "image/svg+xml"
}
