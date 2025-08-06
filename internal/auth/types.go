package auth

import "errors"

type JwtToken string

var (
	ErrorInvalidCredentials = errors.New("Invalid credentials")
	ErrorUserNotFound       = errors.New("User not found")
)
