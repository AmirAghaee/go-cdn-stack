package helper

import "net/http"

type ServiceError struct {
	Code    int
	Message string
}

func (e *ServiceError) Error() string {
	return e.Message
}

func ErrUserExists() *ServiceError {
	return &ServiceError{
		Code:    http.StatusBadRequest,
		Message: "user already exists",
	}
}

func ErrInvalidInput() *ServiceError {
	return &ServiceError{
		Code:    http.StatusBadRequest,
		Message: "invalid inputs",
	}
}

func ErrUnAuthorized() *ServiceError {
	return &ServiceError{
		Code:    http.StatusUnauthorized,
		Message: "invalid credentials",
	}
}

func ErrCdnExists() *ServiceError {
	return &ServiceError{
		Code:    http.StatusBadRequest,
		Message: "cdn already exists",
	}
}
