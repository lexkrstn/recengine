package endpoints

import (
	"errors"
	"net/http"
	"recengine/internal/api/shard/dto"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// Converts validator's FieldError to a message string.
func getFieldErrorMsg(fieldError validator.FieldError) string {
	switch fieldError.Tag() {
	case "required":
		return "This field is required"
	case "lte":
		return "Should be less than " + fieldError.Param()
	case "gte":
		return "Should be greater than " + fieldError.Param()
	}
	return "Unknown error"
}

// Aborts gin handler execution and sends an HTTP response containing the error
// description in JSON format.
func AbortWithBindingErrors(ctx *gin.Context, err error) {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		errors := make([]dto.FieldErrorMsg, len(ve))
		for i, fe := range ve {
			errors[i] = dto.FieldErrorMsg{
				Field:   fe.Field(),
				Message: getFieldErrorMsg(fe),
			}
		}
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error":  "validation",
			"errors": errors,
		})
		return
	}
	ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
		"message": err.Error(),
	})
}

// Aborts gin handler execution and sends an HTTP response containing the error
// description in JSON format.
func AbortWithValidationError(ctx *gin.Context, err error) {
	var ve *dto.ValidationError
	if errors.As(err, &ve) {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error":  "validation",
			"errors": ve.Fields(),
		})
		return
	}
	ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
		"message": err.Error(),
	})
}
