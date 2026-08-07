package validator

import (
	"errors"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func Init() {
	if validate == nil {
		validate = validator.New()
	}
}

func Validate(s interface{}) error {
	Init()
	return validate.Struct(s)
}

func SanitizeString(s string) string {
	return strings.TrimSpace(s)
}

func IsValidEmail(email string) bool {
	Init()
	return validate.Var(email, "email") == nil
}

func ValidatedVar(s interface{}, tag string) error {
	Init()
	return validate.Var(s, tag)
}

// ValidationError returns a friendly message for the first validation error.
func ValidationError(err error) string {
	var verrs validator.ValidationErrors
	if errors.As(err, &verrs) {
		if len(verrs) > 0 {
			return friendlyMessage(verrs[0])
		}
	}
	return "Invalid input. Please check your request."
}

func friendlyMessage(fe validator.FieldError) string {
	field := fe.Field()
	switch fe.Tag() {
	case "required":
		return field + " is required"
	case "email":
		return field + " must be a valid email address"
	case "min":
		return field + " must be at least " + fe.Param() + " characters"
	case "max":
		return field + " must be at most " + fe.Param() + " characters"
	case "oneof":
		return field + " must be one of: " + fe.Param()
	default:
		return field + " is invalid"
	}
}
