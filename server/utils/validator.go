package utils

import (
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"net/http"
)

var validate *validator.Validate

func init() {
	validate = validator.New()
}

// ValidateStruct validates a struct and returns formatted errors
func ValidateStruct(s interface{}) error {
	return validate.Struct(s)
}

// ValidationErrorResponse sends a formatted validation error response
func ValidationErrorResponse(c *gin.Context, err error) {
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		errors := make(map[string]string)
		for _, e := range validationErrors {
			errors[e.Field()] = formatValidationError(e)
		}
		c.JSON(http.StatusBadRequest, gin.H{"errors": errors})
		return
	}
	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
}

func formatValidationError(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return e.Field() + " is required"
	case "email":
		return e.Field() + " must be a valid email"
	case "min":
		return e.Field() + " must be at least " + e.Param() + " characters"
	case "max":
		return e.Field() + " must be at most " + e.Param() + " characters"
	case "gte":
		return e.Field() + " must be greater than or equal to " + e.Param()
	case "lte":
		return e.Field() + " must be less than or equal to " + e.Param()
	default:
		return e.Field() + " is invalid"
	}
}
