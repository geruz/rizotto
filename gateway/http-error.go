package gateway

import "net/http"

type HTTPApiError struct {
	Code      int       `json:"code"`
	Message   string    `json:"message"`
	ErrorCode ErrorCode `json:"errorCode"`
	Details   string    `json:"details"`
}

type (
	PermissionDenied struct{}
	NotFound         struct{}
)

type (
	HTTPError interface {
		ErrorObj() any
		StatusCode() int
	}
	APIError interface {
		Error() string
	}
)

type ErrorCode string

const (
	InternalErrorCode       ErrorCode = "internal_error"
	InvalidRequestErrorCode ErrorCode = "invalid_request"
)

type ValidationError struct {
	FieldErrors []ValidationRecord `json:"fieldErrors"`
	CrossErrors []CrossError       `json:"crossErrors"`
}

func NewValidationError() ValidationError {
	return ValidationError{
		FieldErrors: nil,
		CrossErrors: nil,
	}
}

func (e *ValidationError) AddFieldError(fieldName, message string) {
	e.FieldErrors = append(e.FieldErrors, ValidationRecord{
		ErrorCode: "",
		FieldName: fieldName,
		Message:   message,
		Details:   "",
	})
	e.CrossErrors = nil
}

func (e ValidationError) ErrorObj() any {
	return e
}

func (e ValidationError) StatusCode() int {
	return http.StatusBadRequest
}

type ValidationRecord struct {
	ErrorCode ErrorCode `json:"errorCode"`
	FieldName string    `json:"fieldName"`
	Message   string    `json:"message"`
	Details   string    `json:"details"`
}
type CrossError struct {
	ErrorCode ErrorCode `json:"errorCode"`
	Message   string    `json:"message"`
	Details   string    `json:"details"`
}

func (e HTTPApiError) StatusCode() int {
	return e.Code
}

func (e HTTPApiError) ErrorObj() any {
	return e
}

func NewInternalError() HTTPError {
	return HTTPApiError{
		Code:      http.StatusInternalServerError,
		Message:   "Internal error",
		ErrorCode: InternalErrorCode,
		Details:   "Internal error",
	}
}

func NewInvalidRequest(message string) HTTPError {
	return HTTPApiError{
		Code:      http.StatusBadRequest,
		Message:   message,
		ErrorCode: InvalidRequestErrorCode,
		Details:   "Invalid request",
	}
}

func NewNotFoundError(errorCode ErrorCode, message string, details string) HTTPError {
	return HTTPApiError{
		Code:      http.StatusNotFound,
		Message:   message,
		ErrorCode: errorCode,
		Details:   details,
	}
}

type ConfigurableHTTPError struct {
	OriginalError error
	matched       HTTPError
}

func (c ConfigurableHTTPError) Error() string {
	return c.OriginalError.Error()
}

type ErrorGroup interface {
	IsNotFound() bool
}

func (c ConfigurableHTTPError) IfNotFound(err HTTPError) ConfigurableHTTPError {
	if err == nil {
		return c
	}

	if e, ok := c.OriginalError.(ErrorGroup); ok && e.IsNotFound() {
		c.matched = err
	}

	return c
}

func (c ConfigurableHTTPError) Others(err HTTPError) HTTPError {
	if c.matched != nil {
		return c.matched
	}

	return err
}

func MapError(err error) ConfigurableHTTPError {
	return ConfigurableHTTPError{
		OriginalError: err,
		matched:       nil,
	}
}
