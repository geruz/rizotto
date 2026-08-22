package openapi

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"
	"sync"

	"github.com/danielgtaylor/huma/schema"
	"gopkg.in/yaml.v3"
)

// jsonContentType is the only body media type the documentation deals with.
const jsonContentType = "application/json"

// docMutex guards the document while it is filled in: the route tests that write
// it run in parallel.
//
//nolint:gochecknoglobals // one lock for the document under construction
var docMutex sync.Mutex

type Schema struct {
	Type                 string             `json:"type,omitempty"                 yaml:"type,omitempty"`
	Description          string             `json:"description,omitempty"          yaml:"description,omitempty"`
	Items                *Schema            `json:"items,omitempty"                yaml:"items,omitempty"`
	Properties           map[string]*Schema `json:"properties,omitempty"           yaml:"properties,omitempty"`
	AdditionalProperties any                `json:"additionalProperties,omitempty" yaml:"additionalProperties,omitempty"`
	PatternProperties    map[string]*Schema `json:"patternProperties,omitempty"    yaml:"patternProperties,omitempty"`
	Required             []string           `json:"required,omitempty"             yaml:"required,omitempty"`
	Format               string             `json:"format,omitempty"               yaml:"format,omitempty"`
	Enum                 []any              `json:"enum,omitempty"                 yaml:"enum,omitempty"`
	Default              any                `json:"default,omitempty"              yaml:"default,omitempty"`
	Examples             []any              `json:"examples,omitempty"             yaml:"examples,omitempty"`
	Minimum              *float64           `json:"minimum,omitempty"              yaml:"minimum,omitempty"`
	ExclusiveMinimum     *bool              `json:"exclusiveMinimum,omitempty"     yaml:"exclusiveMinimum,omitempty"`
	Maximum              *float64           `json:"maximum,omitempty"              yaml:"maximum,omitempty"`
	ExclusiveMaximum     *bool              `json:"exclusiveMaximum,omitempty"     yaml:"exclusiveMaximum,omitempty"`
	MultipleOf           float64            `json:"multipleOf,omitempty"           yaml:"multipleOf,omitempty"`
	MinLength            *uint64            `json:"minLength,omitempty"            yaml:"minLength,omitempty"`
	MaxLength            *uint64            `json:"maxLength,omitempty"            yaml:"maxLength,omitempty"`
	Pattern              string             `json:"pattern,omitempty"              yaml:"pattern,omitempty"`
	MinItems             *uint64            `json:"minItems,omitempty"             yaml:"minItems,omitempty"`
	MaxItems             *uint64            `json:"maxItems,omitempty"             yaml:"maxItems,omitempty"`
	UniqueItems          bool               `json:"uniqueItems,omitempty"          yaml:"uniqueItems,omitempty"`
	MinProperties        *uint64            `json:"minProperties,omitempty"        yaml:"minProperties,omitempty"`
	MaxProperties        *uint64            `json:"maxProperties,omitempty"        yaml:"maxProperties,omitempty"`
	AllOf                []*Schema          `json:"allOf,omitempty"                yaml:"allOf,omitempty"`
	AnyOf                []*Schema          `json:"anyOf,omitempty"                yaml:"anyOf,omitempty"`
	OneOf                []*Schema          `json:"oneOf,omitempty"                yaml:"oneOf,omitempty"`
	Not                  *Schema            `json:"not,omitempty"                  yaml:"not,omitempty"`
	Nullable             bool               `json:"nullable,omitempty"             yaml:"nullable,omitempty"`
	ReadOnly             bool               `json:"readOnly,omitempty"             yaml:"readOnly,omitempty"`
	WriteOnly            bool               `json:"writeOnly,omitempty"            yaml:"writeOnly,omitempty"`
	Deprecated           bool               `json:"deprecated,omitempty"           yaml:"deprecated,omitempty"`
	ContentEncoding      string             `json:"contentEncoding,omitempty"      yaml:"contentEncoding,omitempty"`
	Ref                  string             `json:"$ref,omitempty"                 yaml:"$ref,omitempty"`
}

type ServerVariable struct {
	Enum        *[]string `json:"enum,omitempty"        yaml:"enum,omitempty"`
	Default     string    `json:"default,omitempty"     yaml:"default,omitempty"`
	Description string    `json:"description,omitempty" yaml:"description,omitempty"`
}

type ContractInfo struct {
	Name  string `json:"name,omitempty"  yaml:"name,omitempty"`
	Url   string `json:"url,omitempty"   yaml:"url,omitempty"`
	Email string `json:"email,omitempty" yaml:"email,omitempty"`
}
type LicenseInfo struct {
	Name       string `json:"name,omitempty"       yaml:"name,omitempty"`
	Identifier string `json:"identifier,omitempty" yaml:"identifier,omitempty"`
	Url        string `json:"url,omitempty"        yaml:"url,omitempty"`
}
type ServerInfo struct {
	Url         string                    `json:"url,omitempty"         yaml:"url,omitempty"`
	Description string                    `json:"description,omitempty" yaml:"description,omitempty"`
	Variables   map[string]ServerVariable `json:"variables,omitempty"   yaml:"variables,omitempty"`
}
type ExternalDocs struct {
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Url         string `json:"url,omitempty"         yaml:"url,omitempty"`
}
type Tag struct {
	Name         string         `json:"name,omitempty"         yaml:"name,omitempty"`
	Description  string         `json:"description,omitempty"  yaml:"description,omitempty"`
	ExternalDocs []ExternalDocs `json:"externalDocs,omitempty" yaml:"externalDocs,omitempty"`
}

type Info struct {
	Version        string        `json:"version,omitempty"        yaml:"version,omitempty"`
	Title          string        `json:"title,omitempty"          yaml:"title,omitempty"`
	Summary        string        `json:"summary,omitempty"        yaml:"summary,omitempty"`
	Description    string        `json:"description,omitempty"    yaml:"description,omitempty"`
	TermsOfService string        `json:"termsOfService,omitempty" yaml:"termsOfService,omitempty"`
	Contract       *ContractInfo `json:"contract,omitempty"       yaml:"contract,omitempty"`
	License        *LicenseInfo  `json:"license,omitempty"        yaml:"license,omitempty"`
}
type Security struct {
	SecuritySchemes map[string]string `json:"securitySchemes,omitempty" yaml:"securitySchemes,omitempty"`
}
type Content struct {
	Schema *Schema `json:"schema,omitempty" yaml:"schema,omitempty"`
}
type RequestBody struct {
	Description string             `json:"description,omitempty" yaml:"description,omitempty"`
	Content     map[string]Content `json:"content,omitempty"     yaml:"content,omitempty"`
	RequestBody bool               `json:"requestBody"           yaml:"requestBody"`
}
type ResponseObj struct {
	Description string             `json:"description,omitempty" yaml:"description,omitempty"`
	Headers     map[string]any     `json:"headers,omitempty"     yaml:"headers,omitempty"`
	Content     map[string]Content `json:"content,omitempty"     yaml:"content,omitempty"`
	Links       map[string]any     `json:"links,omitempty"       yaml:"links,omitempty"`
}
type Operation struct {
	Description  string              `json:"description,omitempty"  yaml:"description,omitempty"`
	Summary      string              `json:"summary,omitempty"      yaml:"summary,omitempty"`
	Tags         []string            `json:"tags,omitempty"         yaml:"tags,omitempty"`
	ExternalDocs *ExternalDocs       `json:"externalDocs,omitempty" yaml:"externalDocs,omitempty"`
	OperationId  string              `json:"operationId,omitempty"  yaml:"operationId,omitempty"`
	Parameters   []*Parameter        `json:"parameters,omitempty"   yaml:"parameters,omitempty"`
	RequestBody  *RequestBody        `json:"requestBody,omitempty"  yaml:"requestBody,omitempty"`
	Responses    map[int]ResponseObj `json:"responses,omitempty"    yaml:"responses,omitempty"`
	Callbacks    map[string]any      `json:"callbacks,omitempty"    yaml:"callbacks,omitempty"`
	Deprecated   bool                `json:"deprecated,omitempty"   yaml:"deprecated,omitempty"`
	Security     []any               `json:"security,omitempty"     yaml:"security,omitempty"`
	Servers      []ServerInfo        `json:"servers,omitempty"      yaml:"servers,omitempty"`
}
type ExampleObj struct {
	Summary string `json:"summary,omitempty" yaml:"summary,omitempty"`
	Value   any    `json:"value,omitempty"   yaml:"value,omitempty"`
}
type (
	Parameter struct {
		Name        string                `json:"name,omitempty"        yaml:"name,omitempty"`
		In          string                `json:"in,omitempty"          yaml:"in,omitempty"`
		Description string                `json:"description,omitempty" yaml:"description,omitempty"`
		Require     bool                  `json:"required"              yaml:"required"`
		Schema      *Schema               `json:"schema,omitempty"      yaml:"schema,omitempty"`
		Examples    map[string]ExampleObj `json:"examples,omitempty"    yaml:"examples,omitempty"`
	}
	Path struct {
		Description string       `json:"description,omitempty" yaml:"description,omitempty"`
		Summary     string       `json:"summary,omitempty"     yaml:"summary,omitempty"`
		Servers     []ServerInfo `json:"servers,omitempty"     yaml:"servers,omitempty"`
		Parameters  []Parameter  `json:"parameters,omitempty"  yaml:"parameters,omitempty"`

		Get     *Operation `json:"get,omitempty"     yaml:"get,omitempty"`
		Put     *Operation `json:"put,omitempty"     yaml:"put,omitempty"`
		Post    *Operation `json:"post,omitempty"    yaml:"post,omitempty"`
		Delete  *Operation `json:"delete,omitempty"  yaml:"delete,omitempty"`
		Options *Operation `json:"options,omitempty" yaml:"options,omitempty"`
		Head    *Operation `json:"head,omitempty"    yaml:"head,omitempty"`
		Patch   *Operation `json:"patch,omitempty"   yaml:"patch,omitempty"`
		Trace   *Operation `json:"trace,omitempty"   yaml:"trace,omitempty"`
	}
)

type (
	Method        string
	Documentation struct {
		OpenAPI      string           `json:"openapi"                yaml:"openapi"`
		Info         Info             `json:"info"                   yaml:"info"`
		Servers      []ServerInfo     `json:"servers,omitempty"      yaml:"servers,omitempty"`
		Paths        map[string]*Path `json:"paths,omitempty"        yaml:"paths,omitempty"`
		Security     Security         `json:"security"               yaml:"security"`
		Tags         []Tag            `json:"tags,omitempty"         yaml:"tags,omitempty"`
		ExternalDocs *ExternalDocs    `json:"externalDocs,omitempty" yaml:"externalDocs,omitempty"`

		m int
	}
)

func (d *Documentation) Capture() {
	docMutex.Lock()
	defer docMutex.Unlock()

	d.m++
}

const openAPIFilePerm = 0o600

func (d *Documentation) Release() {
	docMutex.Lock()
	defer docMutex.Unlock()

	d.m--
	if d.m == 0 {
		data, _ := yaml.Marshal(d)

		err := os.WriteFile("openapi.yml", data, openAPIFilePerm)
		if err != nil {
			panic(err)
		}
	}
}

func (d *Documentation) String() string {
	docMutex.Lock()
	defer docMutex.Unlock()

	data, err := json.Marshal(d)
	if err != nil {
		return "Marshal error: " + err.Error()
	}

	return string(data)
}

func (d *Documentation) AddRoute(path string) *Path {
	docMutex.Lock()
	defer docMutex.Unlock()

	if d.Paths == nil {
		d.Paths = make(map[string]*Path)
	}

	if _, ok := d.Paths[path]; !ok {
		d.Paths[path] = new(Path)
	}

	return d.Paths[path]
}

var ErrMethodNotSupported = errors.New("method not supported")

func (p *Path) AddOperation(method string) (*Operation, error) {
	docMutex.Lock()
	defer docMutex.Unlock()

	o := new(Operation)

	switch strings.ToLower(method) {
	case "get":
		p.Get = o

		return o, nil
	case "put":
		p.Put = o

		return o, nil
	case "post":
		p.Post = o

		return o, nil
	case "delete":
		p.Delete = o

		return o, nil
	case "options":
		p.Options = o

		return o, nil
	case "head":
		p.Head = o

		return o, nil
	case "patch":
		p.Patch = o

		return o, nil
	case "trace":
		p.Trace = o

		return o, nil
	}

	return nil, ErrMethodNotSupported
}

func findParameter(o *Operation, field reflect.StructField) *Parameter {
	inTagStr := field.Tag.Get("in")
	inParts := strings.Split(inTagStr, ";")
	const namePartNumber = 2

	nameParts := strings.SplitN(inParts[0], "=", namePartNumber)
	if len(nameParts) != namePartNumber {
		return nil
	}

	defValue := ""

	for _, part := range inParts[1:] {
		parts := strings.SplitN(part, "=", namePartNumber)
		if len(parts) == namePartNumber && parts[0] == "default" {
			defValue = parts[1]
		}
	}

	name := nameParts[1]
	in := nameParts[0]

	for _, n := range o.Parameters {
		if n.Name == name {
			return n
		}
	}

	jsonSchema, err := schema.Generate(field.Type)
	if err != nil {
		return nil
	}

	if defValue != "" {
		jsonSchema.Default = defValue
	}

	p := &Parameter{
		Name:        name,
		In:          in,
		Description: field.Name,
		Require:     defValue == "",
		Schema:      convertSchema(jsonSchema, nil),
		Examples:    make(map[string]ExampleObj),
	}
	o.Parameters = append(o.Parameters, p)

	return p
}

func (o *Operation) AddRequest(desc string, value any) error {
	docMutex.Lock()
	defer docMutex.Unlock()

	if value == nil {
		return nil
	}

	typeOfVal := reflect.TypeOf(value)

	val := reflect.ValueOf(value)

	bodyFields := []reflect.StructField{}
	bodyExample := map[string]any{}

	for i := range val.NumField() {
		field := typeOfVal.Field(i)

		// a field without an "in" directive is read from the request body
		if field.Tag.Get("in") == "" {
			bodyFields = append(bodyFields, field)
			bodyExample[bodyFieldName(field)] = val.Field(i).Interface()

			continue
		}

		p := findParameter(o, field)
		if p == nil {
			continue
		}

		p.Examples[desc] = ExampleObj{
			Summary: desc,
			Value:   val.Field(i).Interface(),
		}
	}

	if len(bodyFields) == 0 {
		return nil
	}

	return o.addRequestBody(desc, bodyFields, bodyExample)
}

// bodyFieldName is the name the field has in the json body.
func bodyFieldName(field reflect.StructField) string {
	name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
	if name == "" || name == "-" {
		return field.Name
	}

	return name
}

func (o *Operation) AddResponse(desc string, statusCode int, response any) error {
	docMutex.Lock()
	defer docMutex.Unlock()

	if o.Responses == nil {
		o.Responses = make(map[int]ResponseObj)
	}

	jsonSchema, err := schema.Generate(reflect.TypeOf(response))
	if err != nil {
		return err
	}

	o.Responses[statusCode] = ResponseObj{
		Description: desc,
		Headers:     nil,
		Content: map[string]Content{
			jsonContentType: {
				Schema: convertSchema(jsonSchema, response),
			},
		},
		Links: nil,
	}

	return nil
}

// addRequestBody documents the fields the handler reads from the json body.
func (o *Operation) addRequestBody(desc string, fields []reflect.StructField, example map[string]any) error {
	properties := make(map[string]*Schema, len(fields))
	required := make([]string, 0, len(fields))

	for _, field := range fields {
		jsonSchema, err := schema.Generate(field.Type)
		if err != nil {
			return err
		}

		name := bodyFieldName(field)
		properties[name] = convertSchema(jsonSchema, nil)

		if !strings.Contains(field.Tag.Get("json"), ",omitempty") {
			required = append(required, name)
		}
	}

	if o.RequestBody == nil {
		o.RequestBody = &RequestBody{
			Description: desc,
			Content:     map[string]Content{},
			RequestBody: true,
		}
	}

	content := o.RequestBody.Content[jsonContentType]
	if content.Schema == nil {
		//nolint:exhaustruct_v5 // an object schema only needs its properties
		content.Schema = &Schema{
			Type:                 "object",
			Properties:           properties,
			Required:             required,
			AdditionalProperties: false,
		}
	}

	content.Schema.Examples = append(content.Schema.Examples, example)
	o.RequestBody.Content[jsonContentType] = content

	return nil
}

type Description string

func convertSchemaMap(jsonSchemas map[string]*schema.Schema) map[string]*Schema {
	if jsonSchemas == nil {
		return nil
	}

	schemas := make(map[string]*Schema, len(jsonSchemas))
	for key, jsonSchema := range jsonSchemas {
		schemas[key] = convertSchema(jsonSchema, nil)
	}

	return schemas
}

func convertSchemaList(jsonSchemas []*schema.Schema) []*Schema {
	if jsonSchemas == nil {
		return nil
	}

	schemas := make([]*Schema, len(jsonSchemas))
	for i, jsonSchema := range jsonSchemas {
		schemas[i] = convertSchema(jsonSchema, nil)
	}

	return schemas
}

func convertSchema(jsonSchema *schema.Schema, example any) *Schema {
	if jsonSchema == nil {
		return nil
	}

	var examples []any = nil
	if example != nil {
		examples = []any{example}
	}

	return &Schema{
		Type:                 jsonSchema.Type,
		Description:          jsonSchema.Description,
		Items:                convertSchema(jsonSchema.Items, nil),
		Properties:           convertSchemaMap(jsonSchema.Properties),
		AdditionalProperties: jsonSchema.AdditionalProperties,
		PatternProperties:    convertSchemaMap(jsonSchema.PatternProperties),
		Required:             jsonSchema.Required,
		Format:               jsonSchema.Format,
		Enum:                 jsonSchema.Enum,
		Default:              jsonSchema.Default,
		Examples:             examples,
		Minimum:              jsonSchema.Minimum,
		ExclusiveMinimum:     jsonSchema.ExclusiveMinimum,
		Maximum:              jsonSchema.Maximum,
		ExclusiveMaximum:     jsonSchema.ExclusiveMaximum,
		MultipleOf:           jsonSchema.MultipleOf,
		MinLength:            jsonSchema.MinLength,
		MaxLength:            jsonSchema.MaxLength,
		Pattern:              jsonSchema.Pattern,
		MinItems:             jsonSchema.MinItems,
		MaxItems:             jsonSchema.MaxItems,
		UniqueItems:          jsonSchema.UniqueItems,
		MinProperties:        jsonSchema.MinProperties,
		MaxProperties:        jsonSchema.MaxProperties,
		AllOf:                convertSchemaList(jsonSchema.AllOf),
		AnyOf:                convertSchemaList(jsonSchema.AnyOf),
		OneOf:                convertSchemaList(jsonSchema.OneOf),
		Not:                  convertSchema(jsonSchema.Not, nil),
		Nullable:             jsonSchema.Nullable,
		ReadOnly:             jsonSchema.ReadOnly,
		WriteOnly:            jsonSchema.WriteOnly,
		Deprecated:           jsonSchema.Deprecated,
		ContentEncoding:      jsonSchema.ContentEncoding,
		Ref:                  jsonSchema.Ref,
	}
}

func (d Description) AddOperationDetails(o *Operation) {
	o.Description = string(d)
}
