package apperror

import (
	"fmt"

	"github.com/rzfd/expand/internal/pkg/logging"
)

type Error struct {
	Status  int
	Code    string
	Message string
	Details any
	Err     error
}

func (e *Error) Error() string {
	logging.FromContext(nil).Info().Str("code", e.Code).Int("status", e.Status).Msg("app error string started")
	if e.Err == nil {
		logging.FromContext(nil).Info().Str("code", e.Code).Msg("app error string completed without cause")
		return e.Message
	}
	result := fmt.Sprintf("%s: %v", e.Message, e.Err)
	logging.FromContext(nil).Info().Str("code", e.Code).Msg("app error string completed with cause")
	return result
}

func (e *Error) Unwrap() error {
	logging.FromContext(nil).Info().Str("code", e.Code).Msg("app error unwrap")
	return e.Err
}

func New(status int, code, message string) *Error {
	logging.FromContext(nil).Info().Int("status", status).Str("code", code).Msg("app error new")
	return &Error{
		Status:  status,
		Code:    code,
		Message: message,
	}
}

func WithDetails(err *Error, details any) *Error {
	logging.FromContext(nil).Info().Str("code", err.Code).Msg("app error with details")
	err.Details = details
	return err
}

func Wrap(status int, code, message string, err error) *Error {
	logging.FromContext(nil).Info().Int("status", status).Str("code", code).Msg("app error wrap")
	return &Error{
		Status:  status,
		Code:    code,
		Message: message,
		Err:     err,
	}
}
