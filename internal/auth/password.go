package auth

import (
	"errors"

	"golang.org/x/crypto/bcrypt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/auth/usecase"
)

type BcryptPasswordService struct{}

func (BcryptPasswordService) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func (BcryptPasswordService) Verify(password, hash string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return usecase.ErrPasswordMismatch
	}
	return err
}
