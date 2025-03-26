package mcp

import (
	"fmt"
)

// ErrorCode represents an error code for domain-specific errors
type ErrorCode string

// Error codes for different error scenarios
const (
	// ErrCodeInvalidParameter indicates an invalid parameter was provided
	ErrCodeInvalidParameter ErrorCode = "INVALID_PARAMETER"
	// ErrCodeNotFound indicates a requested resource was not found
	ErrCodeNotFound ErrorCode = "NOT_FOUND"
	// ErrCodeClientError indicates an error in the Portainer client
	ErrCodeClientError ErrorCode = "CLIENT_ERROR"
	// ErrCodeServerError indicates a server-side error
	ErrCodeServerError ErrorCode = "SERVER_ERROR"
	// ErrCodeUnauthorized indicates an authorization error
	ErrCodeUnauthorized ErrorCode = "UNAUTHORIZED"
)

// DomainError represents a domain-specific error with a code and message
type DomainError struct {
	Code    ErrorCode
	Message string
	Err     error
}

// Error returns the string representation of the error
func (e *DomainError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error
func (e *DomainError) Unwrap() error {
	return e.Err
}

// NewInvalidParameterError creates a new invalid parameter error
func NewInvalidParameterError(message string, err error) *DomainError {
	return &DomainError{
		Code:    ErrCodeInvalidParameter,
		Message: message,
		Err:     err,
	}
}

// NewNotFoundError creates a new not found error
func NewNotFoundError(message string, err error) *DomainError {
	return &DomainError{
		Code:    ErrCodeNotFound,
		Message: message,
		Err:     err,
	}
}

// NewClientError creates a new client error
func NewClientError(message string, err error) *DomainError {
	return &DomainError{
		Code:    ErrCodeClientError,
		Message: message,
		Err:     err,
	}
}

// NewServerError creates a new server error
func NewServerError(message string, err error) *DomainError {
	return &DomainError{
		Code:    ErrCodeServerError,
		Message: message,
		Err:     err,
	}
}

// NewUnauthorizedError creates a new unauthorized error
func NewUnauthorizedError(message string, err error) *DomainError {
	return &DomainError{
		Code:    ErrCodeUnauthorized,
		Message: message,
		Err:     err,
	}
}