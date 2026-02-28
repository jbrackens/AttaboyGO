package domain

import "fmt"

// AppError is the base domain error type.
type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"-"`
	Cause   error  `json:"-"`
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error { return e.Cause }

// Standard domain error constructors.

func ErrNotFound(entity, id string) *AppError {
	return &AppError{Code: "NOT_FOUND", Message: fmt.Sprintf("%s %s not found", entity, id), Status: 404}
}

func ErrConflict(msg string) *AppError {
	return &AppError{Code: "CONFLICT", Message: msg, Status: 409}
}

func ErrValidation(msg string) *AppError {
	return &AppError{Code: "VALIDATION_ERROR", Message: msg, Status: 400}
}

func ErrUnauthorized(msg string) *AppError {
	return &AppError{Code: "UNAUTHORIZED", Message: msg, Status: 401}
}

func ErrForbidden(msg string) *AppError {
	return &AppError{Code: "FORBIDDEN", Message: msg, Status: 403}
}

func ErrInsufficientBalance() *AppError {
	return &AppError{Code: "INSUFFICIENT_BALANCE", Message: "insufficient balance", Status: 400}
}

func ErrIdempotent(existingTxID string) *AppError {
	return &AppError{Code: "IDEMPOTENT", Message: fmt.Sprintf("transaction already exists: %s", existingTxID), Status: 200}
}

func ErrAccountLocked(msg string) *AppError {
	return &AppError{Code: "ACCOUNT_LOCKED", Message: msg, Status: 429}
}

func ErrInternal(msg string, cause error) *AppError {
	return &AppError{Code: "INTERNAL_ERROR", Message: msg, Status: 500, Cause: cause}
}
