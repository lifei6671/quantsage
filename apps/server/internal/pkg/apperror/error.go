package apperror

import "errors"

// AppError is the canonical business error wrapper.
type AppError struct {
	Code int
	Err  error
}

// New wraps an error with a business code.
func New(code int, err error) *AppError {
	return &AppError{
		Code: code,
		Err:  err,
	}
}

func (e *AppError) Error() string {
	if e == nil {
		return ""
	}
	if e.Err != nil {
		return e.Err.Error()
	}

	errmsg, _ := MessageOf(e.Code)
	return errmsg
}

// Unwrap exposes the underlying cause for errors.As / errors.Is.
func (e *AppError) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.Err
}

// CodeOf returns the business code for an error.
func CodeOf(err error) int {
	if err == nil {
		return CodeOK
	}

	var appErr *AppError
	if errors.As(err, &appErr) && appErr != nil {
		return appErr.Code
	}

	return CodeInternal
}

// MessageOf maps a business code to errmsg and toast text.
func MessageOf(code int) (errmsg string, toast string) {
	msg, ok := messages[code]
	if !ok {
		msg = messages[CodeInternal]
	}

	return msg.Errmsg, msg.Toast
}
