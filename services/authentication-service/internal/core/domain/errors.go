package domain

import "errors"

// Определяем переменные-ошибки, которые могут быть возвращены из Use Cases.
var (
	ErrUserNotFound      = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrEmailInUse        = errors.New("email already in use")
	ErrTokenInvalid      = errors.New("invalid jwt token")
)