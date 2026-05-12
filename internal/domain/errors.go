package domain

import "errors"

type ValidationErr struct{ msg string }

func (e ValidationErr) Error() string { return e.msg }

func validationError(msg string) ValidationErr { return ValidationErr{msg} }

func IsValidation(err error) bool {
	var ve ValidationErr
	return errors.As(err, &ve)
}

var ErrNotFound = errors.New("not found")
var ErrConflict = errors.New("conflict")
