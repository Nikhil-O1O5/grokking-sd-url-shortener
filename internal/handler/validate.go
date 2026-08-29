package handler

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

func validateStruct(s any) error {
	err := validate.Struct(s)
	if err == nil {
		return nil
	}

	var errs []string
	for _, e := range err.(validator.ValidationErrors) {
		errs = append(errs, fieldError(e))
	}
	return fmt.Errorf("%s", strings.Join(errs, ", "))
}

func fieldError(e validator.FieldError) string {
	field := strings.ToLower(e.Field())
	switch e.Tag() {
	case "required":
		return field + " is required"
	case "email":
		return field + " must be a valid email"
	case "min":
		return fmt.Sprintf("%s must be at least %s characters", field, e.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s characters", field, e.Param())
	default:
		return field + " is invalid"
	}
}
