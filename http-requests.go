package rizotto

import (
	"errors"
	"net/http"
	"reflect"

	"github.com/go-playground/validator/v10"
	// "github.com/shopspring/decimal".
)

var validate *validator.Validate = validator.New()

/*func init() {
	if validate == nil {
		return
	}
	validate.RegisterCustomTypeFunc(func(field reflect.Value) interface{} {
		if valuer, ok := field.Interface().(decimal.Decimal); ok {
			return valuer.String()
		}
		return nil
	}, decimal.Decimal{})
	if err := validate.RegisterValidation("dgte", ValidateDecimalGreatOrEq); err != nil {
		panic(err)
	}
	if err := validate.RegisterValidation("maxdec", ValidateMaxDecimals); err != nil {
		panic(err)
	}


func ValidateMaxDecimals(fl validator.FieldLevel) bool {
	data, ok := fl.Field().Interface().(string)
	if !ok {
		return false
	}
	fieldValue, err := decimal.NewFromString(data)
	if err != nil {
		return false
	}
	maxPlaces, err := decimal.NewFromString(fl.Param())
	if err != nil {
		return false
	}
	return int64(fieldValue.Exponent()) >= -maxPlaces.IntPart()
}

func ValidateDecimalGreatOrEq(fl validator.FieldLevel) bool {
	data, ok := fl.Field().Interface().(string)
	if !ok {
		return false
	}
	value, err := decimal.NewFromString(data)
	if err != nil {
		return false
	}
	baseValue, err := decimal.NewFromString(fl.Param())
	if err != nil {
		return false
	}
	return value.GreaterThanOrEqual(baseValue)
}

}*/

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

func validateRequestObject[TRequest any](reqParams TRequest) HTTPError {
	err := validate.Struct(reqParams)
	if err == nil {
		return nil
	}

	{
		var validationErr validator.ValidationErrors
		var validationErr1 *validator.InvalidValidationError
		switch {
		case errors.As(err, &validationErr):
			validationError := NewValidationError()
			rt := reflect.TypeOf(reqParams)
			if rt.Kind() == reflect.Pointer {
				rt = rt.Elem()
			}
			for _, err := range validationErr {
				name := err.Field()
				if str, isFound := rt.FieldByName(name); isFound {
					if n, ok := tryGetValidationFieldNameFrom(str); ok {
						name = n
					}
				}
				validationError.AddFieldError(name, err.Error())
			}

			return validationError
		case errors.As(err, &validationErr1):
			return NewInvalidRequest(validationErr1.Error())
		default:
			return NewInvalidRequest(err.Error())
		}
	}
}

func tryGetValidationFieldNameFrom(str reflect.StructField) (string, bool) {
	jsonName := str.Tag.Get("json")
	if jsonName != "" {
		return jsonName, true
	}

	inName := str.Tag.Get("in")
	if inName != "" {
		return inName, true
	}

	return "", false
}
