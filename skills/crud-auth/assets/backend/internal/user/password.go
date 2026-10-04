package user

import (
	"sync"

	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

func HashPassword(password string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	return string(h), err
}

// CheckPassword compares in constant time. An empty hash (SSO-only user) never matches.
func CheckPassword(hash, password string) bool {
	if hash == "" {
		_ = bcrypt.CompareHashAndPassword(dummyHash(), []byte(password))
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// SimulatePasswordCheck burns the same time as a real check so unknown emails can't be detected by timing.
func SimulatePasswordCheck(password string) {
	_ = bcrypt.CompareHashAndPassword(dummyHash(), []byte(password))
}

var dummyHash = sync.OnceValue(func() []byte {
	h, _ := bcrypt.GenerateFromPassword([]byte("timing-equalizer"), bcryptCost)
	return h
})
