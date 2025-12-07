package domain

import (
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// User - основная доменная сущность
type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string 
	Role 		 string
	CreatedAt    time.Time
}

// Claims - это данные, которые мы "зашиваем" в JWT токен.
type Claims struct {
	UserID uuid.UUID
	Email  string
	Role   string
}

// NewUser создает нового пользователя. Хэширование пароля происходит здесь.
func NewUser(email, password string) (*User, error) {
	// Хэшируем пароль с использованием bcrypt.
	// bcrypt.DefaultCost - это хороший баланс между скоростью и безопасностью.
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	return &User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: string(hashedPassword),
		Role:         "user",
		CreatedAt:    time.Now().UTC(),
	}, nil
}

// CheckPassword сравнивает предоставленный пароль с хэшем, хранящимся у пользователя.
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}