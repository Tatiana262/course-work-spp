package token_adapter

import (
	"authentication-service/internal/contextkeys"
	"authentication-service/internal/core/domain"
	"authentication-service/internal/core/port"
	"context"
	"errors"
	"fmt"
	// "log"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// TokenService - реализация TokenServicePort для JWT.
type TokenService struct {
	// Секретный ключ для подписи токенов. Должен быть длинным и сложным.
	// Хранится в конфиге и передается при создании сервиса.
	signingKey []byte
}

func NewTokenService(signingKey string) (*TokenService, error) {
	if signingKey == "" {
		return nil, fmt.Errorf("JWT signing key cannot be empty")
	}
	return &TokenService{signingKey: []byte(signingKey)}, nil
}

// jwtCustomClaims - это наша реализация стандартных claims JWT.
type jwtCustomClaims struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	Role   string    `json:"role"`
	jwt.RegisteredClaims
}

// GenerateToken создает новый JWT токен.
func (s *TokenService) GenerateToken(ctx context.Context, user *domain.User, ttl time.Duration) (string, error) {
	logger := contextkeys.LoggerFromContext(ctx)
	serviceLogger := logger.WithFields(port.Fields{
		"component": "TokenService",
		"method":    "GenerateToken",
		"user_id":   user.ID.String(),
	})
	
	serviceLogger.Info("Generating new token.", port.Fields{"ttl": ttl.String()})
	claims := &jwtCustomClaims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "auth-service", // Имя вашего сервиса
		},
	}

	// Создаем токен с нашими claims и методом подписи HS256
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Подписываем токен нашим секретным ключом
	signedToken, err := token.SignedString(s.signingKey)
	if err != nil {
		serviceLogger.Error("Failed to sign token", err, nil)
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	serviceLogger.Info("Token generated successfully.", nil)
	return signedToken, nil
}

// ValidateToken проверяет токен.
func (s *TokenService) ValidateToken(ctx context.Context, tokenString string) (*domain.Claims, error) {
	logger := contextkeys.LoggerFromContext(ctx)
	serviceLogger := logger.WithFields(port.Fields{
		"component": "TokenService",
		"method":    "ValidateToken",
	})

	serviceLogger.Info("Attempting to validate token.", nil)

	token, err := jwt.ParseWithClaims(tokenString, &jwtCustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Проверяем, что метод подписи - HS256, как мы и ожидали
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			alg := token.Header["alg"]
			serviceLogger.Error("Unexpected signing method detected", fmt.Errorf("algorithm %v is not HS256", alg), port.Fields{"algorithm": alg})
			return nil, fmt.Errorf("unexpected signing method: %v", alg)
		}
		return s.signingKey, nil
	})

	if err != nil {
		// Проверяем, была ли ошибка ИМЕННО из-за истечения срока
		if errors.Is(err, jwt.ErrTokenExpired) {
			// Здесь token.Valid будет false, но claims можно прочитать
			if claims, ok := token.Claims.(*jwtCustomClaims); ok {
				 serviceLogger.Warn("Token has expired", port.Fields{"user_id": claims.UserID.String(), "email": claims.Email})
			}  else {
				serviceLogger.Warn("An expired token could not be parsed to claims", nil)
			}	
		} else {
			// Это была другая, более серьезная ошибка (например, подделка)
			serviceLogger.Error("Invalid token format or signature", err, nil)		
		}
		return nil, domain.ErrTokenInvalid
	}

	// проверка token.Valid только если отключить стандартную валидацию
	if claims, ok := token.Claims.(*jwtCustomClaims); ok && token.Valid {
		serviceLogger.Info("Token validated successfully.", port.Fields{
			"user_id": claims.UserID.String(),
			"email":   claims.Email,
			"role":    claims.Role,
		})
		
		return &domain.Claims{
			UserID: claims.UserID,
			Email:  claims.Email,
			Role:   claims.Role,
		}, nil
	}
	
	serviceLogger.Error("Token was parsed without error, but claims type assertion failed", nil, nil)
	return nil, domain.ErrTokenInvalid
}