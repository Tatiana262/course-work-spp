package rest

// RegisterRequest - тело запроса для регистрации.
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string `json:"token"`
	// UserID string `json:"user_id"`
	// Role   string `json:"role"` 
}

// LoginRequest - тело запроса для входа.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// ValidateTokenRequest - тело запроса для валидации токена.
type ValidateTokenRequest struct {
	Token string `json:"token"`
}

// ValidateTokenResponse - тело ответа при успешной валидации.
type ValidateTokenResponse struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"` 
}

// ErrorResponse - стандартная структура для ответа с ошибкой.
type ErrorResponse struct {
	Error string `json:"error"`
}