package apperror

import (
	"fmt"
)

type Error struct {
	Status  int
	Code    string
	Message string
	Details any
	Err     error
}

func (e *Error) Error() string {
	if e.Err == nil {
		return e.Message
	}
	return fmt.Sprintf("%s: %v", e.Message, e.Err)
}

func (e *Error) Unwrap() error {
	return e.Err
}

func New(status int, code, message string) *Error {
	return &Error{
		Status:  status,
		Code:    code,
		Message: message,
	}
}

func WithDetails(err *Error, details any) *Error {
	err.Details = details
	return err
}

func Wrap(status int, code, message string, err error) *Error {
	return &Error{
		Status:  status,
		Code:    code,
		Message: message,
		Err:     err,
	}
}
