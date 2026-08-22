package validators

import (
	"context"
	"errors"
	"reflect"

	"github.com/geruz/rizotto/logger"
	"github.com/go-playground/validator/v10"
	"github.com/shopspring/decimal"
)

const messageKey = "message"

var validate *validator.Validate = validator.New()

var validationMessageBuilders = map[string]func(name, param string) string{
	"email":    func(name, _ string) string { return name + " is not a valid email address" },
	"required": func(name, _ string) string { return name + " is a required property" },
	"oneof":    func(name, param string) string { return name + " should be one of the following: " + param },
	"lte":      func(name, param string) string { return name + " should be less or equal: " + param },
	"gte":      func(name, param string) string { return name + " should be greater or equal: " + param },
	"gt":       func(name, param string) string { return name + " should be greater: " + param },
	"lt":       func(name, param string) string { return name + " should be less: " + param },
	"min":      func(name, param string) string { return name + " should be greater or equal: " + param },
	"dgte":     func(name, param string) string { return name + " should be greater or equal: " + param },
	"maxdec": func(name, param string) string {
		return name + " should have no more than " + param + " decimal places"
	},
	"max": func(name, param string) string { return name + " should be at most " + param + " characters long" },
}

func validationMessage(name string, err validator.FieldError) (string, bool) {
	build, ok := validationMessageBuilders[err.Tag()]
	if !ok {
		return "", false
	}

	return build(name, err.Param()), true
}

func init() {
	if validate == nil {
		return
	}

	validate.RegisterCustomTypeFunc(func(field reflect.Value) any {
		if valuer, ok := reflect.TypeAssert[decimal.Decimal](field); ok {
			return valuer.String()
		}

		return nil
	}, decimal.Decimal{})

	err := validate.RegisterValidation("dgte", ValidateDecimalGreatOrEq)
	if err != nil {
		panic(err)
	}

	err = validate.RegisterValidation("maxdec", ValidateMaxDecimals)
	if err != nil {
		panic(err)
	}
}

func ValidateHTTPRequest(ctx context.Context, request any) any {
	validationError := validate.Struct(request)
	if validationError == nil {
		return nil
	}

	rt := reflect.TypeOf(request)
	if rt.Kind() == reflect.Pointer {
		rt = rt.Elem()
	}

	invalidValidationError := new(validator.InvalidValidationError)
	ok := errors.As(validationError, &invalidValidationError)
	if ok {
		return map[string]any{
			messageKey: []string{
				invalidValidationError.Error(),
			},
		}
	}
	var validationErrors validator.ValidationErrors
	ok = errors.As(validationError, &validationErrors)
	if !ok {
		return map[string]any{
			messageKey: []string{"Unknown validation error"},
		}
	}
	messages := []string{}
	for _, err := range validationErrors {
		name := err.Field()
		if str, isFound := rt.FieldByName(name); isFound {
			if n, ok := tryGetValidationFieldNameFrom(str); ok {
				name = n
			}
		}

		if msg, ok := validationMessage(name, err); ok {
			messages = append(messages, msg)
		} else {
			logger.Error(ctx, "Unknown validation error", err)
		}
	}

	return map[string]any{
		messageKey: messages,
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

func ValidateMaxDecimals(fl validator.FieldLevel) bool {
	data, ok := reflect.TypeAssert[string](fl.Field())
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
	data, ok := reflect.TypeAssert[string](fl.Field())
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
