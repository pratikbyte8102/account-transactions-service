package service

import "errors"

var (
	ErrConflict   = errors.New("conflict")
	ErrNotFound   = errors.New("not found")
	ErrValidation = errors.New("validation error")
)

type ValidationError struct {
	Message string
}

func (e ValidationError) Error() string {
	return e.Message
}

func (e ValidationError) Unwrap() error {
	return ErrValidation
}
