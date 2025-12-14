package usecase

import (
	"authentication-service/internal/contextkeys"
	"authentication-service/internal/core/domain"
	"authentication-service/internal/core/port"
	"context"
	"fmt"
	"time"
)

type LoginUserUseCase struct {
	userRepo    port.UserRepositoryPort
	tokenSvc    port.TokenServicePort
	accessTokenTTL time.Duration
}

func NewLoginUserUseCase(userRepo port.UserRepositoryPort, tokenSvc port.TokenServicePort, accessTokenTTL time.Duration) *LoginUserUseCase {
	return &LoginUserUseCase{
		userRepo:    userRepo,
		tokenSvc:    tokenSvc,
		accessTokenTTL: accessTokenTTL,
	}
}

func (uc *LoginUserUseCase) Execute(ctx context.Context, email, password string) (*domain.User, string, error) {
	logger := contextkeys.LoggerFromContext(ctx)
	ucLogger := logger.WithFields(port.Fields{
		"use_case": "LoginUser",
		"email":    email,
	})
	ucLogger.Info("Use case started: attempting to login user", nil)

	// Находим пользователя по email
	user, err := uc.userRepo.FindByEmail(ctx, email)
	if err != nil {
		// ошибка БД
		ucLogger.Error("Repository failed to find user by email", err, nil)
		return nil, "", fmt.Errorf("internal server error: %w", err) 
	}
	if user == nil {
		// пользователь не найден
		ucLogger.Warn("Login failed: user not found", nil)
		return nil, "", domain.ErrUserNotFound
	}

	ucLogger = ucLogger.WithFields(port.Fields{"user_id": user.ID.String()})

	// Проверяем пароль
	if !user.CheckPassword(password) {
		ucLogger.Warn("Login failed: invalid credentials", nil)
		return nil, "", domain.ErrInvalidCredentials
	}

	// Генерируем токен
	token, err := uc.tokenSvc.GenerateToken(ctx, user, uc.accessTokenTTL)
	if err != nil {
		ucLogger.Error("Failed to generate token after successful login", err, nil)
		return nil, "", err
	}

	ucLogger.Info("Use case finished: user logged in successfully", nil)
	return user, token, nil
}