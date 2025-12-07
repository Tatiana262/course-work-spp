package rest

import (
	"authentication-service/internal/contextkeys"
	"authentication-service/internal/core/domain"
	"authentication-service/internal/core/port"
	"authentication-service/internal/core/port/usecases_port"
	"encoding/json"
	"errors"
	// "log"
	"net/http"
)

// AuthHandlers реализует интерфейс Handlers.
type AuthHandlers struct {
	registerUC    usecases_port.RegisterUserUseCasePort
	loginUC       usecases_port.LoginUserUseCasePort
	validateUC    usecases_port.ValidateTokenUseCasePort
}

// NewAuthHandlers - конструктор.
func NewAuthHandlers(registerUC usecases_port.RegisterUserUseCasePort, 
	loginUC usecases_port.LoginUserUseCasePort,
	validateUC usecases_port.ValidateTokenUseCasePort) *AuthHandlers {
	return &AuthHandlers{
		registerUC:    registerUC,
		loginUC:       loginUC,
		validateUC:    validateUC,
	}
}

// Register обрабатывает POST /register
func (h *AuthHandlers) Register(w http.ResponseWriter, r *http.Request) {

	logger := contextkeys.LoggerFromContext(r.Context()).WithFields(port.Fields{"handler": "Register"})
	
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("Failed to decode register request body", port.Fields{"error": err.Error()})
		WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Простая валидация
	if req.Email == "" || req.Password == "" {
		logger.Warn("Email and password are required", nil)
		WriteJSONError(w, http.StatusBadRequest, "Email and password are required")
		return
	}

	// Обогащаем логгер данными запроса (без пароля!)
	handlerLogger := logger.WithFields(port.Fields{
		"email":   req.Email,
	})
	handlerLogger.Info("Processing register request", nil)

	user, token, err := h.registerUC.Execute(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, domain.ErrEmailInUse) {
			handlerLogger.Warn("Registration failed: email already in use", nil)
            WriteJSONError(w, http.StatusConflict, err.Error())
            return
        }
		handlerLogger.Error("Register use case failed with an unexpected error", err, nil)
		WriteJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	handlerLogger.Info("User registered successfully", port.Fields{"user_id": user.ID})

	response := AuthResponse{
		Token:  token,
		UserID: user.ID.String(),
		Role:   user.Role,
	}
	RespondWithJSON(w, http.StatusCreated, response)
}

// Login обрабатывает POST /login
func (h *AuthHandlers) Login(w http.ResponseWriter, r *http.Request) {
	logger := contextkeys.LoggerFromContext(r.Context()).WithFields(port.Fields{"handler": "Login"})

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("Failed to decode login request body", port.Fields{"error": err.Error()})
		WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	handlerLogger := logger.WithFields(port.Fields{"email": req.Email})
	handlerLogger.Info("Processing login request", nil)

	user, token, err := h.loginUC.Execute(r.Context(), req.Email, req.Password)
	if err != nil {
		// Ошибка "invalid credentials" - это 401 Unauthorized
		if errors.Is(err, domain.ErrInvalidCredentials) || errors.Is(err, domain.ErrUserNotFound) {
			handlerLogger.Warn("Login failed: invalid credentials", nil)
            WriteJSONError(w, http.StatusUnauthorized, err.Error())
            return
        }
		// Любая другая ошибка - это 500
		handlerLogger.Error("Login use case failed with an unexpected error", err, nil)
		WriteJSONError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	handlerLogger.Info("User logged in successfully", port.Fields{"user_id": user.ID})

	RespondWithJSON(w, http.StatusOK, AuthResponse{
		Token: token,
		UserID: user.ID.String(),
		Role:   user.Role,
	})
}

// ValidateToken обрабатывает POST /validate
func (h *AuthHandlers) ValidateToken(w http.ResponseWriter, r *http.Request) {
	logger := contextkeys.LoggerFromContext(r.Context())

	handlerLogger := logger.WithFields(port.Fields{"handler": "ValidateToken"})
	handlerLogger.Info("Processing token validation request", nil)

	var req ValidateTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handlerLogger.Warn("Failed to decode validation request body", port.Fields{"error": err.Error()})
		WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	claims, err := h.validateUC.Execute(r.Context(), req.Token)
	if err != nil {
		handlerLogger.Warn("Token validation failed", port.Fields{"error": err.Error()})
		WriteJSONError(w, http.StatusUnauthorized, "Invalid or expired token")
		return
	}

	handlerLogger.Info("Token validated successfully", port.Fields{
		"user_id": claims.UserID,
		"role":    claims.Role,
	})

	RespondWithJSON(w, http.StatusOK, ValidateTokenResponse{
		UserID: claims.UserID.String(),
		Email:  claims.Email,
		Role:   claims.Role,
	})
}