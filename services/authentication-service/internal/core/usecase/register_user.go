package usecase

import (
	"authentication-service/internal/contextkeys"
	"authentication-service/internal/core/domain"
	"authentication-service/internal/core/port"
	"context"
	"fmt"
	// "log"
	"time"
)

type RegisterUserUseCase struct {
    userRepo port.UserRepositoryPort
    tokenSvc port.TokenServicePort 
    accessTokenTTL time.Duration
}

func NewRegisterUserUseCase(userRepo port.UserRepositoryPort, tokenSvc port.TokenServicePort, accessTokenTTL time.Duration) *RegisterUserUseCase {
	return &RegisterUserUseCase{
		userRepo:    userRepo,
		tokenSvc:    tokenSvc,
		accessTokenTTL: accessTokenTTL,
	}
}

func (uc *RegisterUserUseCase) Execute(ctx context.Context, email, password string) (*domain.User, string, error) {
	logger := contextkeys.LoggerFromContext(ctx)
	ucLogger := logger.WithFields(port.Fields{
		"use_case": "RegisterUser",
		"email":    email,
	})

	ucLogger.Info("Use case started: attempting to register user", nil)
	
	existingUser, err := uc.userRepo.FindByEmail(ctx, email)
	if err != nil {
		ucLogger.Error("Repository failed while checking for existing email", err, nil)
		return nil, "", fmt.Errorf("internal server error: %w", err) 
	}
	if existingUser != nil {
		ucLogger.Warn("Registration failed: email already in use", nil)
		return nil, "", domain.ErrEmailInUse // Кастомная ошибка
	}

	// Создаем нового пользователя (хэширование пароля происходит внутри NewUser)
	user, err := domain.NewUser(email, password)
	if err != nil {
		ucLogger.Error("Failed to create new user domain object", err, nil)
		return nil, "", err
	}

	ucLogger = ucLogger.WithFields(port.Fields{"user_id": user.ID.String()}) 

	// Сохраняем пользователя в репозиторий
	if err := uc.userRepo.Create(ctx, user); err != nil {
		ucLogger.Error("Repository failed to create user", err, nil)
		return nil, "", err
	}

	// Сразу после создания пользователя генерируем для него токен
	token, err := uc.tokenSvc.GenerateToken(ctx, user, uc.accessTokenTTL)
	if err != nil {
		ucLogger.Error("Failed to generate token after successful registration", err, nil)
		return nil, "", err
	}
 
	ucLogger.Info("Use case finished: user registered successfully", nil)
	return user, token, nil
}