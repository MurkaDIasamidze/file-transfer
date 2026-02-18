package utils

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// AppError is a structured application error
type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}

func (e *AppError) Error() string {
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

func NewError(code int, msg string) *AppError {
	return &AppError{Code: code, Message: msg}
}

func NewFieldError(code int, field, msg string) *AppError {
	return &AppError{Code: code, Message: msg, Field: field}
}

// Respond writes an AppError as a JSON response
func Respond(c *fiber.Ctx, err error) error {
	var ae *AppError
	if errors.As(err, &ae) {
		return c.Status(ae.Code).JSON(fiber.Map{
			"error": ae.Message,
			"field": ae.Field,
		})
	}
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
		"error": err.Error(),
	})
}

// ValidateStruct uses reflect to check all string/ptr fields are non-zero
// and returns a list of missing required fields.
func ValidateStruct(v any) []string {
	var missing []string
	rv := reflect.ValueOf(v)
	rt := reflect.TypeOf(v)

	// dereference pointer
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
		rt = rt.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return missing
	}

	for i := 0; i < rv.NumField(); i++ {
		field := rt.Field(i)
		fval := rv.Field(i)

		// only validate fields tagged with `validate:"required"`
		tag := field.Tag.Get("validate")
		if !strings.Contains(tag, "required") {
			continue
		}

		jsonTag := field.Tag.Get("json")
		name := strings.Split(jsonTag, ",")[0]
		if name == "" {
			name = field.Name
		}

		switch fval.Kind() {
		case reflect.String:
			if strings.TrimSpace(fval.String()) == "" {
				missing = append(missing, name)
			}
		case reflect.Ptr, reflect.Interface:
			if fval.IsNil() {
				missing = append(missing, name)
			}
		case reflect.Int, reflect.Int64:
			if fval.Int() == 0 {
				missing = append(missing, name)
			}
		}
	}
	return missing
}

// BindAndValidate parses body into dst and validates required fields
func BindAndValidate(c *fiber.Ctx, dst any) error {
	if err := c.BodyParser(dst); err != nil {
		return NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if missing := ValidateStruct(dst); len(missing) > 0 {
		return NewError(fiber.StatusUnprocessableEntity,
			fmt.Sprintf("missing required fields: %s", strings.Join(missing, ", ")))
	}
	return nil
}