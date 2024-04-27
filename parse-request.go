package rizotto

import (
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/ggicci/httpin"
	"github.com/ggicci/httpin/core"
)

type urlParamFunc func(r *http.Request, key string) string

func init() {
	core.RegisterDirective(
		"path",
		core.NewDirectivePath((&httpURLParamExtractor{
			URLParam: func(r *http.Request, key string) string {
				return r.PathValue(key)
			},
		}).Execute),
		true,
	)
}

type httpURLParamExtractor struct {
	URLParam urlParamFunc
}

func (chi *httpURLParamExtractor) Execute(rtm *core.DirectiveRuntime) error {
	req := rtm.GetRequest()
	kvs := make(map[string][]string)

	for _, key := range rtm.Directive.Argv {
		value := chi.URLParam(req, key)
		if value != "" {
			kvs[key] = []string{value}
		}
	}

	extractor := &core.FormExtractor{
		Runtime: rtm,
		Form: multipart.Form{
			Value: kvs,
			File:  nil,
		},
		KeyNormalizer: nil,
	}

	return extractor.Extract()
}

func requestObjBuilder[TRequest any]() func(w http.ResponseWriter, r *http.Request) (TRequest, HTTPError) {
	engine, err := httpin.New(new(TRequest))
	if err != nil {
		panic(err)
	}

	return func(w http.ResponseWriter, req *http.Request) (TRequest, HTTPError) {
		parsedQuery, err := engine.Decode(req)
		if err != nil {
			var req TRequest

			return req, NewInvalidRequest(err.Error())
		}

		reqObj, ok := parsedQuery.(*TRequest)
		if !ok {
			var req TRequest

			return req, NewInvalidRequest("Invalid  cast of parsed request object")
		}

		if parsingError := parseBody(req, reqObj); parsingError != nil {
			return *reqObj, parsingError
		}

		return *reqObj, validateRequestObject(reqObj)
	}
}

func parseBody[TRequest any](req *http.Request, requestObj *TRequest) HTTPError {
	if req.Body == nil {
		return nil
	}

	jsonBody, err := getBodyJSON(req)
	if err != nil {
		return NewInvalidRequest(err.Error())
	}

	if len(jsonBody) == 0 {
		return nil
	}

	err = json.Unmarshal(jsonBody, requestObj)
	if err != nil {
		return NewInvalidRequest(err.Error())
	}

	return nil
}

func getBodyJSON(req *http.Request) ([]byte, error) {
	if isFormBody(req) {
		err := req.ParseForm()
		if err != nil {
			return nil, err
		}

		return formBodyToRawJSON(req)
	}

	return io.ReadAll(req.Body)
}

func isFormBody(r *http.Request) bool {
	return r.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
}

func formBodyToRawJSON(r *http.Request) ([]byte, error) {
	singleValueBody := make(map[string]string, len(r.PostForm))

	for key, value := range r.PostForm {
		if len(value) == 0 {
			singleValueBody[key] = ""
		} else {
			singleValueBody[key] = value[0]
		}
	}

	return json.Marshal(singleValueBody)
}
