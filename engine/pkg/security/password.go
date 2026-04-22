package security

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

func HashPassword(password string) (string, error) {
	if len(password) < 6 {
		return "", fmt.Errorf("password must be at least 6 characters")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}

func CheckPassword(password string, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
