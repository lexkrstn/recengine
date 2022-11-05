package dto

// JSON format of validation errors.
type FieldErrorMsg struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ValidationError struct {
	fields []FieldErrorMsg
}

func NewValidationError() *ValidationError {
	return &ValidationError{
		fields: make([]FieldErrorMsg, 0, 5),
	}
}

func NewValidationErrorField(field string, err error) *ValidationError {
	ve := NewValidationError()
	ve.AddError(field, err)
	return ve
}

func AddValidationErrorField(ve *ValidationError, field string, err error) *ValidationError {
	if ve == nil {
		return NewValidationErrorField(field, err)
	}
	ve.AddError(field, err)
	return ve
}

func (err *ValidationError) Fields() []FieldErrorMsg {
	return err.fields
}

func (err *ValidationError) Error() string {
	return "Validation error"
}

func (err *ValidationError) AddError(field string, fieldError error) {
	err.AddMessage(field, fieldError.Error())
}

func (err *ValidationError) AddMessage(field string, message string) {
	err.fields = append(err.fields, FieldErrorMsg{
		Field:   field,
		Message: message,
	})
}
