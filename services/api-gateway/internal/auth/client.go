package auth

import (
	"api-gateway/internal/contextkeys"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Claims - структура, описывающая полезную нагрузку токена.
// Она должна совпадать с `port.Claims` из authentication-service.
type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
}

// DTO для запроса на валидацию.
type validateTokenRequest struct {
	Token string `json:"token"`
}

// Client - клиент для взаимодействия с authentication-service.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient - конструктор клиента.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{},
	}
}

// ValidateToken отправляет токен в authentication-service и возвращает claims.
func (c *Client) ValidateToken(ctx context.Context, token string) (*Claims, error) {
	// 1. Формируем тело запроса
	reqBody, err := json.Marshal(validateTokenRequest{Token: token})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal validation request: %w", err)
	}

	// 2. Создаем HTTP POST-запрос
	url := c.baseURL + "/api/v1/auth/validate"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create validation request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	traceID := contextkeys.TraceIDFromContext(ctx)
	if traceID != "" {
		req.Header.Set("X-Trace-ID", traceID)
	}

	// 3. Выполняем запрос
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send validation request to auth service: %w", err)
	}
	defer resp.Body.Close()

	// 4. Проверяем статус-код ответа
	if resp.StatusCode != http.StatusOK {
		// Если auth-service вернул 401, значит токен невалиден.
		// Мы не считаем это внутренней ошибкой, а просто пробрасываем "невалидность".
		if resp.StatusCode == http.StatusUnauthorized {
			return nil, fmt.Errorf("token is invalid or expired")
		}
		// Любой другой код - это уже проблема на стороне auth-service.
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("auth service returned non-200 status: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	// 5. Декодируем успешный ответ
	var claims Claims
	if err := json.NewDecoder(resp.Body).Decode(&claims); err != nil {
		return nil, fmt.Errorf("failed to decode validation response: %w", err)
	}

	return &claims, nil
}