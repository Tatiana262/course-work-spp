package usecase

import (
	"authentication-service/internal/contextkeys"
	"authentication-service/internal/core/domain"
	"authentication-service/internal/core/port"
	"context"
)

type ValidateTokenUseCase struct {
	tokenSvc port.TokenServicePort
}

func NewValidateTokenUseCase(tokenSvc port.TokenServicePort) *ValidateTokenUseCase {
	return &ValidateTokenUseCase{tokenSvc: tokenSvc}
}

func (uc *ValidateTokenUseCase) Execute(ctx context.Context, tokenString string) (*domain.Claims, error) {
	logger := contextkeys.LoggerFromContext(ctx)
	ucLogger := logger.WithFields(port.Fields{
		"use_case": "ValidateToken",
	})
	
	ucLogger.Info("Use case started: validating token", nil)

	claims, err := uc.tokenSvc.ValidateToken(ctx, tokenString)
	if err != nil {
		ucLogger.Warn("Token validation failed", port.Fields{"error": err.Error()})
		return nil, err
	}
	
	ucLogger.Info("Use case finished: token validated successfully", port.Fields{
		"user_id": claims.UserID.String(),
		"role":    claims.Role,
	})
	return claims, nil
}