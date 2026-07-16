package bb

import (
	"errors"
	"net/http"
)

type baseError struct {
	StatusCode int    `json:"statusCode"`
	ErrorCode  string `json:"errorCode"`
}
type (
	ServiceError interface {
		Code() string
		IsNotFound() bool
		Error() string
	}
)

func (e *baseError) Code() string {
	return e.ErrorCode
}

func (e *baseError) IsNotFound() bool {
	return e.ErrorCode == "not_found"
}

type NotImplementedError struct {
	baseError

	PrivateDetails string `json:"privateDetails"`
}

func NewNotImplementedError(service string, method string) *NotImplementedError {
	return &NotImplementedError{
		baseError: baseError{
			StatusCode: http.StatusNotImplemented,
			ErrorCode:  "not_implemented",
		},
		PrivateDetails: "method " + method + " is not implemented",
	}
}

func (e *NotImplementedError) Error() string {
	return e.PrivateDetails
}

type WrongContractError struct {
	baseError

	PrivateDetails string `json:"privateDetails"`
}

func NewWrongContractError(service string, method string) *WrongContractError {
	return &WrongContractError{
		baseError: baseError{
			StatusCode: http.StatusBadRequest,
			ErrorCode:  "wrong_contract",
		},
		PrivateDetails: "method " + method + " has wrong contract",
	}
}

func (e *WrongContractError) Error() string {
	return e.PrivateDetails
}

type NotFoundError struct {
	baseError

	PrivateDetails string `json:"privateDetails"`
}

func IsNotFoundError(err error) bool {
	var v *NotFoundError

	return errors.As(err, &v)
}

func NewNotFoundError(message string) *NotFoundError {
	return &NotFoundError{
		baseError: baseError{
			StatusCode: http.StatusNotFound,
			ErrorCode:  "not_found",
		},
		PrivateDetails: message,
	}
}

func (e *NotFoundError) Error() string {
	return e.PrivateDetails
}

type UnauthorizedError struct {
	baseError

	PrivateDetails string `json:"privateDetails"`
}

func IsUnauthorizedError(err error) bool {
	var v *UnauthorizedError

	return errors.As(err, &v)
}

func NewUnauthorizedError(message string) *UnauthorizedError {
	return &UnauthorizedError{
		baseError: baseError{
			StatusCode: http.StatusUnauthorized,
			ErrorCode:  "unauthorized",
		},
		PrivateDetails: message,
	}
}

func (e *UnauthorizedError) Error() string {
	return e.PrivateDetails
}

type InternalError struct {
	baseError

	originalError  error
	PrivateDetails string `json:"privateDetails"`
}

func NewInternalError(message string, err error) *InternalError {
	privateDetails := message
	if err != nil {
		privateDetails += ": " + err.Error()
	}

	return &InternalError{
		baseError: baseError{
			StatusCode: http.StatusInternalServerError,
			ErrorCode:  "internal_error",
		},
		originalError:  err,
		PrivateDetails: privateDetails,
	}
}

func (e *InternalError) Unwrap() error {
	return e.originalError
}

func (e *InternalError) Error() string {
	return e.PrivateDetails
}

type ValidationError struct {
	baseError

	PrivateDetails string `json:"privateDetails"`

	PublicErrorCode string  `json:"publicErrorCode"`
	PublicMessage   *string `json:"publicMessage,omitempty"`
}

var ValidationErrorCode = "validation_error"

func NewValidationError[TCode ~string](publicErrorCode TCode, message string) *ValidationError {
	return &ValidationError{
		baseError: baseError{
			StatusCode: http.StatusBadRequest,
			ErrorCode:  ValidationErrorCode,
		},
		PrivateDetails:  message,
		PublicErrorCode: string(publicErrorCode),
		PublicMessage:   nil,
	}
}

func NewInvalidRequestError(message string) *ValidationError {
	return &ValidationError{
		baseError: baseError{
			StatusCode: http.StatusBadRequest,
			ErrorCode:  "validation_error",
		},
		PrivateDetails:  message,
		PublicErrorCode: "invalid_request",
		PublicMessage:   nil,
	}
}
func NewPublicValidationError(privateDetails string, publicDetails string) *ValidationError {
	return &ValidationError{
		baseError: baseError{
			StatusCode: http.StatusBadRequest,
			ErrorCode:  "validation_error",
		},
		PrivateDetails:  privateDetails,
		PublicErrorCode: "public_validation_error",
		PublicMessage:   &publicDetails,
	}
}

func (e *ValidationError) Error() string {
	return e.PrivateDetails
}

func IsValidationErrorWithCode[TErrorCode ~string](err error, code TErrorCode) bool {
	var v *ValidationError
	if ok := errors.As(err, &v); ok {
		if TErrorCode(v.PublicErrorCode) == code {
			return true
		}
	}

	return false
}
