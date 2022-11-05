package dto

// Basic http error response.
type Error struct {
	Message string `json:"message"`
}

func FromError(err error) Error {
	return Error{
		Message: err.Error(),
	}
}
